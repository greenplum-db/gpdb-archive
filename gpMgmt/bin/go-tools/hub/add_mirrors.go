package hub

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/greenplum"
)

func (s *Server) AddMirrors(req *idl.AddMirrorsRequest, stream idl.Hub_AddMirrorsServer) error {
	hubStream := NewHubStream(stream)
	hubStream.StreamLogMsg("Starting to add mirrors to the cluster")

	// Make sure all agents are up and listening for requests
	err := s.DialAllAgents()
	if err != nil {
		return utils.LogAndReturnError(err)
	}

	conn, err := greenplum.GetCoordinatorConn(req.CoordinatorDataDir, "", true)
	if err != nil {
		return utils.LogAndReturnError(err)
	}
	defer conn.Close()

	gparray, err := greenplum.NewGpArrayFromCatalog(conn)
	if err != nil {
		return utils.LogAndReturnError(err)
	}

	// Check if the number of primary and mirror segments are equal
	hubStream.StreamLogMsg("Checking if the number of primary segments and the number of mirrors to add are equal")
	if len(gparray.GetPrimarySegments()) != len(req.Mirrors) {
		return utils.LogAndReturnError(fmt.Errorf("number of mirrors %d is not equal to the number of primaries %d present in the cluster", len(req.Mirrors), len(gparray.GetPrimarySegments())))
	}

	// Check if the cluster already has mirrors, if yes error out
	hubStream.StreamLogMsg("Checking if the cluster already has mirrors")
	if gparray.HasMirrors() {
		return utils.LogAndReturnError(fmt.Errorf("cannot add mirrors, the cluster is already configured with mirrors"))
	}

	// Register the mirrors to the gp_segment_configuration
	hubStream.StreamLogMsg("Starting to register mirror segments with the coordinator")
	err = greenplum.RegisterMirrorSegments(req.Mirrors, conn)
	if err != nil {
		return utils.LogAndReturnError(err)
	}
	hubStream.StreamLogMsg("Successfully registered the mirror segments with the coordinator")

	// Build the new gparray and validate it
	gparray, err = greenplum.NewGpArrayFromCatalog(conn)
	if err != nil {
		return utils.LogAndReturnError(err)
	}

	// Update the pg_hba.conf on the primary segments - Agent RPC
	hubStream.StreamLogMsg("Starting to modify the pg_hba.conf on the primary segments to add mirror entries")
	err = s.UpdatePgHbaConfWithMirrorEntries(gparray, req.Mirrors, req.HbaHostnames)
	if err != nil {
		return utils.LogAndReturnError(err)
	}
	hubStream.StreamLogMsg("Successfully modified the pg_hba.conf on the primary segments")

	// Run pg_basebackup aon the mirror hosts - Agent RPC
	hubStream.StreamLogMsg("Creating mirror segments")
	err = s.CreateMirrorSegments(&hubStream, gparray, req.Mirrors)
	if err != nil {
		return utils.LogAndReturnError(err)
	}
	hubStream.StreamLogMsg("Successfully created mirror segments")

	// Start the segment - Agent RPC
	hubStream.StreamLogMsg("Starting up the mirror segments")
	err = s.StartMirrorSegments(req.Mirrors)
	if err != nil {
		return utils.LogAndReturnError(err)
	}
	hubStream.StreamLogMsg("Successfully started the mirror segments")

	// Run FTS
	hubStream.StreamLogMsg("Triggering FTS probe")
	err = greenplum.TriggerFtsProbe(req.CoordinatorDataDir)
	if err != nil {
		return utils.LogAndReturnError(err)
	}

	hubStream.StreamLogMsg("Mirror segments have been added")
	hubStream.StreamLogMsg("Data synchronization might be in progress and will continue in the background")
	hubStream.StreamLogMsg("Use 'gpstate -s' to check the resynchronization progress")

	return nil
}

func (s *Server) CreateMirrorSegments(stream hubStreamer, gparray *greenplum.GpArray, mirrorSegs []*idl.Segment) error {
	mirrorHostToSegPairMap := make(map[string][]*greenplum.SegmentPair)
	for _, seg := range mirrorSegs {
		pair, err := gparray.GetSegmentPairForContent(int(seg.Contentid))
		if err != nil {
			return err
		}

		mirrorHostToSegPairMap[pair.Mirror.Hostname] = append(mirrorHostToSegPairMap[pair.Mirror.Hostname], pair)
	}

	progressLabel := "Initializing mirror segments:"
	progressTotal := len(mirrorSegs)
	stream.StreamProgressMsg(progressLabel, progressTotal)

	request := func(conn *Connection) error {
		var wg sync.WaitGroup

		pairs := mirrorHostToSegPairMap[conn.Hostname]
		errs := make(chan error, len(pairs))
		for _, pair := range pairs {
			pair := pair
			wg.Add(1)

			go func(pair *greenplum.SegmentPair) {
				defer wg.Done()

				gplog.Debug(fmt.Sprintf("Starting to create mirror segment: %v", *pair.Mirror))
				req := &idl.PgBasebackupRequest{
					TargetDir:           pair.Mirror.DataDir,
					SourceHost:          pair.Primary.Hostname,
					SourcePort:          int32(pair.Primary.Port),
					CreateSlot:          true,
					TargetDbid:          int32(pair.Mirror.Dbid),
					WriteRecoveryConf:   true,
					ReplicationSlotName: constants.ReplicationSlotName,
				}
				_, err := conn.AgentClient.PgBasebackup(context.Background(), req)
				if err != nil {
					errs <- utils.FormatGrpcError(err)
					return
				}
				gplog.Debug("Successfully ran pg_basebackup on segment with data directory %s on host %s", pair.Primary.DataDir, pair.Primary.Hostname)

				gplog.Debug("Starting to modify the postgresql.conf for segment with data directory %s on host %s with port value %d", pair.Mirror.DataDir, pair.Mirror.Hostname, pair.Mirror.Port)
				_, err = conn.AgentClient.UpdatePgConf(context.Background(), &idl.UpdatePgConfRequest{
					Pgdata: pair.Mirror.DataDir,
					Params: map[string]string{
						"port": strconv.Itoa(pair.Mirror.Port),
					},
					Overwrite: true,
				})
				if err != nil {
					errs <- err
				} else {
					gplog.Debug("Successfully modified the postgresql.conf for segment with data directory %s on host %s with port value %d", pair.Mirror.DataDir, pair.Mirror.Hostname, pair.Mirror.Port)
					stream.StreamProgressMsg(progressLabel, progressTotal)
					gplog.Debug(fmt.Sprintf("Successfully created mirror segment: %v", *pair.Mirror))
				}
			}(pair)
		}

		wg.Wait()
		close(errs)

		var err error
		for e := range errs {
			err = errors.Join(err, e)
		}

		return err
	}

	return ExecuteRPC(s.Conns, request)
}

func (s *Server) StartMirrorSegments(mirrorSegs []*idl.Segment) error {
	hostToSegMap := make(map[string][]*idl.Segment)
	for _, seg := range mirrorSegs {
		hostToSegMap[seg.HostName] = append(hostToSegMap[seg.HostName], seg)
	}

	request := func(conn *Connection) error {
		var wg sync.WaitGroup

		segs := hostToSegMap[conn.Hostname]
		errs := make(chan error, len(segs))
		for _, seg := range segs {
			seg := seg
			wg.Add(1)

			go func(seg *idl.Segment) {
				defer wg.Done()

				req := &idl.StartSegmentRequest{
					DataDir: seg.DataDirectory,
					Wait:    true,
					Options: "-c gp_role=execute",
				}
				_, err := conn.AgentClient.StartSegment(context.Background(), req)
				if err != nil {
					errs <- utils.FormatGrpcError(err)
				}
			}(seg)
		}

		wg.Wait()
		close(errs)

		var err error
		for e := range errs {
			err = errors.Join(err, e)
		}

		return err
	}

	return ExecuteRPC(s.Conns, request)
}
