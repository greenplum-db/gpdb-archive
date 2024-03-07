package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/postgres"
)

// PgBasebackup is an agent RPC implementation that executes the pg_basebackup command.
// It redirects its output to a file, which is cleaned up if the command executes successfully.
// Additionally, it manages the removal of any previously created replication slots if a new one needs to be created.
func (s *Server) PgBasebackup(ctx context.Context, req *idl.PgBasebackupRequest) (*idl.PgBasebackupResponse, error) {
	pgBasebackupCmd := &postgres.PgBasebackup{
		TargetDir:           req.TargetDir,
		SourceHost:          req.SourceHost,
		SourcePort:          int(req.SourcePort),
		CreateSlot:          req.CreateSlot,
		ForceOverwrite:      req.ForceOverwrite,
		TargetDbid:          int(req.TargetDbid),
		WriteRecoveryConf:   req.WriteRecoveryConf,
		ReplicationSlotName: req.ReplicationSlotName,
		ExcludePaths:        req.ExcludePaths,
	}

	if pgBasebackupCmd.CreateSlot {
		// Drop any previously created slot to avoid error when creating a new slot with the same name.
		err := postgres.DropSlotIfExists(pgBasebackupCmd.SourceHost, pgBasebackupCmd.SourcePort, pgBasebackupCmd.ReplicationSlotName)
		if err != nil {
			return &idl.PgBasebackupResponse{}, fmt.Errorf("failed to drop replication slot %s: %w", pgBasebackupCmd.ReplicationSlotName, err)
		}
	}

	// TODO Check if the directory is empty if ForceOverwrite is false

	pgBasebackupLog := filepath.Join(s.LogDir, fmt.Sprintf("pg_basebackup.%s.dbid%d.out", time.Now().Format("20060102_150405"), req.TargetDbid))
	out, err := utils.RunGpCommandAndRedirectOutput(pgBasebackupCmd, s.GpHome, pgBasebackupLog)
	if err != nil {
		return &idl.PgBasebackupResponse{}, fmt.Errorf("executing pg_basebackup: %s, logfile: %s, %w", out, pgBasebackupLog, err)
	}
	os.Remove(pgBasebackupLog)

	return &idl.PgBasebackupResponse{}, nil
}
