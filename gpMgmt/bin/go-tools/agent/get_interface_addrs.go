package agent

import (
	"context"

	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
)

// GetInterfaceAddrs is the implementation of agent RPC which returns the
// network interface addresses for the host without the loopback address.
func (s *Server) GetInterfaceAddrs(ctx context.Context, req *idl.GetInterfaceAddrsRequest) (*idl.GetInterfaceAddrsResponse, error) {
	addrs, err := utils.GetHostAddrsNoLoopback()
	if err != nil {
		return &idl.GetInterfaceAddrsResponse{}, err
	}

	return &idl.GetInterfaceAddrsResponse{Addrs: addrs}, nil
}
