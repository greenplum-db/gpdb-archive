package hub

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils/greenplum"
)

// UpdatePgHbaConfWithMirrorEntries updates the pg_hba.conf file on the primary segments
// with the details of its corresponding mirror segment pair. The hbaHostname parameter
// determines whether to use hostnames or IP addresses in the pg_hba.conf file.
func (s *Server) UpdatePgHbaConfWithMirrorEntries(gparray *greenplum.GpArray, mirrorSegs []*idl.Segment, hbaHostname bool) error {
	primaryHostToSegPairMap := make(map[string][]*greenplum.SegmentPair)
	for _, seg := range mirrorSegs {
		pair, err := gparray.GetSegmentPairForContent(int(seg.Contentid))
		if err != nil {
			return err
		}

		primaryHostToSegPairMap[pair.Primary.Hostname] = append(primaryHostToSegPairMap[pair.Primary.Hostname], pair)
	}

	request := func(conn *Connection) error {
		var wg sync.WaitGroup

		pairs := primaryHostToSegPairMap[conn.Hostname]
		errs := make(chan error, len(pairs))
		for _, pair := range pairs {
			pair := pair
			wg.Add(1)

			go func(pair *greenplum.SegmentPair) {
				var addrs []string
				var err error
				defer wg.Done()

				if hbaHostname {
					addrs = []string{pair.Primary.Address, pair.Mirror.Address}
				} else {
					primaryAddrs, err := s.GetInterfaceAddrs(pair.Primary.Hostname)
					if err != nil {
						errs <- err
						return
					}

					mirrorAddrs, err := s.GetInterfaceAddrs(pair.Mirror.Hostname)
					if err != nil {
						errs <- err
						return
					}

					addrs = append(primaryAddrs, mirrorAddrs...)
				}

				_, err = conn.AgentClient.UpdatePgHbaConfAndReload(context.Background(), &idl.UpdatePgHbaConfRequest{
					Pgdata:      pair.Primary.DataDir,
					Addrs:       addrs,
					Replication: true,
				})
				if err != nil {
					errs <- err
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

// GetInterfaceAddrs returns the interface addresses for a given host.
// It retrieves the interface addresses by executing an RPC call to the agent client.
func (s *Server) GetInterfaceAddrs(host string) ([]string, error) {
	conns := getConnForHosts(s.Conns, []string{host})

	var addrs []string
	request := func(conn *Connection) error {
		resp, err := conn.AgentClient.GetInterfaceAddrs(context.Background(), &idl.GetInterfaceAddrsRequest{})
		if err != nil {
			return fmt.Errorf("failed to get interface addresses for host %s: %w", conn.Hostname, err)
		}
		addrs = resp.Addrs

		return nil
	}

	err := ExecuteRPC(conns, request)

	return addrs, err
}
