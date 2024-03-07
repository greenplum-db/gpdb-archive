package postgres

import (
	"os/exec"
	"path/filepath"

	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/utils"
)

const (
	initdbUtility        = "initdb"
	pgCtlUtility         = "pg_ctl"
	postgresUtility      = "postgres"
	pgbasebackupUtility  = "pg_basebackup"
	pgControlDataUtility = "pg_controldata"
)

type Initdb struct {
	PgData        string `flag:"--pgdata"`
	Encoding      string `flag:"--encoding"`
	Locale        string `flag:"--locale"`
	LcCollate     string `flag:"--lc-collate"`
	LcCtype       string `flag:"--lc-ctype"`
	LcMessages    string `flag:"--lc-messages"`
	LcMonetory    string `flag:"--lc-monetary"`
	LcNumeric     string `flag:"--lc-numeric"`
	LcTime        string `flag:"--lc-time"`
	DataChecksums bool   `flag:"--data-checksums"`
}

func (cmd *Initdb) BuildExecCommand(gpHome string) *exec.Cmd {
	utility := utils.GetGpUtilityPath(gpHome, initdbUtility)
	args := utils.GenerateArgs(cmd)

	return utils.System.ExecCommand(utility, args...)
}

type PgCtlStart struct {
	PgData  string `flag:"--pgdata"`
	Timeout int    `flag:"--timeout"`
	Wait    bool   `flag:"--wait"`
	NoWait  bool   `flag:"--no-wait"`
	Logfile string `flag:"--log"`
	Options string `flag:"--options"`
	// TODO: Check if we need to set GPSESSID and GP_ERA env variables
}

func (cmd *PgCtlStart) BuildExecCommand(gpHome string) *exec.Cmd {
	utility := utils.GetGpUtilityPath(gpHome, pgCtlUtility)

	if cmd.Timeout == 0 {
		cmd.Timeout = constants.DefaultStartTimeout
	}

	// TODO: Consider changing this to the log_directory GUC or gpAdminLogs
	if cmd.Logfile == "" {
		cmd.Logfile = filepath.Join(cmd.PgData, constants.DefaultPostgresLogDir, "startup.log")
	}

	args := append([]string{"start"}, utils.GenerateArgs(cmd)...)

	return utils.System.ExecCommand(utility, args...)
}

type PgCtlStop struct {
	PgData  string `flag:"--pgdata"`
	Timeout int    `flag:"--timeout"`
	Wait    bool   `flag:"--wait"`
	NoWait  bool   `flag:"--no-wait"`
	Mode    string `flag:"--mode"`
}

func (cmd *PgCtlStop) BuildExecCommand(gpHome string) *exec.Cmd {
	utility := utils.GetGpUtilityPath(gpHome, pgCtlUtility)
	args := append([]string{"stop"}, utils.GenerateArgs(cmd)...)

	return utils.System.ExecCommand(utility, args...)
}

type PgCtlReload struct {
	PgData string `flag:"--pgdata"`
}

func (cmd *PgCtlReload) BuildExecCommand(gpHome string) *exec.Cmd {
	utility := utils.GetGpUtilityPath(gpHome, pgCtlUtility)
	args := append([]string{"reload"}, utils.GenerateArgs(cmd)...)

	return utils.System.ExecCommand(utility, args...)
}

type Postgres struct {
	GpVersion bool `flag:"--gp-version"`
}

func (cmd *Postgres) BuildExecCommand(gpHome string) *exec.Cmd {
	utililty := utils.GetGpUtilityPath(gpHome, postgresUtility)
	args := utils.GenerateArgs(cmd)
	return utils.System.ExecCommand(utililty, args...)
}

type PgBasebackup struct {
	TargetDir           string `flag:"--pgdata"`
	SourceHost          string `flag:"--host"`
	SourcePort          int    `flag:"--port"`
	CreateSlot          bool   `flag:"--create-slot"`
	ForceOverwrite      bool   `flag:"--force-overwrite"`
	TargetDbid          int    `flag:"--target-gp-dbid"`
	WriteRecoveryConf   bool   `flag:"--write-recovery-conf"`
	ReplicationSlotName string
	ExcludePaths        []string
}

func (cmd *PgBasebackup) BuildExecCommand(gphome string) *exec.Cmd {
	utility := utils.GetGpUtilityPath(gphome, pgbasebackupUtility)
	args := []string{"--checkpoint", "fast", "--no-verify-checksums", "--progress", "--verbose"}

	args = append(args, utils.GenerateArgs(cmd)...)

	if cmd.ReplicationSlotName != "" {
		args = append(args, "--slot", cmd.ReplicationSlotName, "--wal-method", "stream")
	} else {
		args = append(args, "--wal-method", "fetch")
	}

	defaultExcludeList := []string{"./db_dumps", "./promote", "./db_analyze"}
	if len(cmd.ExcludePaths) == 0 {
		cmd.ExcludePaths = defaultExcludeList
	}
	for _, path := range cmd.ExcludePaths {
		args = append(args, "--exclude", path)
	}

	return utils.System.ExecCommand(utility, args...)
}

type PgControlData struct {
	PgData string `flag:"--pgdata"`
}

func (cmd *PgControlData) BuildExecCommand(gphome string) *exec.Cmd {
	utility := utils.GetGpUtilityPath(gphome, pgControlDataUtility)
	args := utils.GenerateArgs(cmd)

	return utils.System.ExecCommand(utility, args...)
}
