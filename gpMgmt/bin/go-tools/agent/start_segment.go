package agent

import (
	"context"
	"fmt"

	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/postgres"
)

/*
StartSegment implements agent RPC to start the segment.
Input: data-directory, wait and timeout.
Makes a call to pg_ctl start command
*/
func (s *Server) StartSegment(ctx context.Context, in *idl.StartSegmentRequest) (*idl.StartSegmentReply, error) {
	pgCtlStartOptions := postgres.PgCtlStart{
		PgData:  in.DataDir,
		Wait:    in.Wait,
		Timeout: int(in.Timeout),
		Options: in.Options,
	}
	out, err := utils.RunGpCommand(&pgCtlStartOptions, s.GpHome)
	if err != nil {
		return &idl.StartSegmentReply{}, utils.LogAndReturnError(fmt.Errorf("executing pg_ctl start: %s, logfile: %s, %w", out, pgCtlStartOptions.Logfile, err))
	}

	return &idl.StartSegmentReply{}, nil
}
