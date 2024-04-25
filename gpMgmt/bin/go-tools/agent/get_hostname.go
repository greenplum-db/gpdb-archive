package agent

import (
	"context"
	"fmt"

	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
)

func (s *Server) GetHostName(ctx context.Context, request *idl.GetHostNameRequest) (*idl.GetHostNameReply, error) {
	hostname, err := utils.System.GetHostName()
	if err != nil {

		return &idl.GetHostNameReply{}, utils.LogAndReturnError(fmt.Errorf("error getting hostname:%v", err))
	}

	return &idl.GetHostNameReply{Hostname: hostname}, nil
}
