package agent

import (
	"context"
	"fmt"

	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/postgres"
)

// UpdatePgHbaConf is agent RPC implementation which updates the segment pg_hba.conf
// with the given address list and then reloads the segment with pg_ctl reload.
func (s *Server) UpdatePgHbaConfAndReload(ctx context.Context, req *idl.UpdatePgHbaConfRequest) (*idl.UpdatePgHbaConfResponse, error) {
	err := postgres.UpdateSegmentPgHbaConf(req.Pgdata, req.Addrs, req.Replication)
	if err != nil {
		return &idl.UpdatePgHbaConfResponse{}, fmt.Errorf("updating pg_hba.conf: %w", err)
	}

	pgCtlReloadCmd := &postgres.PgCtlReload{
		PgData: req.Pgdata,
	}
	out, err := utils.RunGpCommand(pgCtlReloadCmd, s.GpHome)
	if err != nil {
		return &idl.UpdatePgHbaConfResponse{}, fmt.Errorf("executing pg_ctl reload: %s, %w", out, err)
	}

	return &idl.UpdatePgHbaConfResponse{}, nil
}
