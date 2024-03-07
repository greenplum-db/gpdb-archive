package greenplum

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/postgres"
)

var newDBConnFromEnvironment = dbconn.NewDBConnFromEnvironment

func GetPostgresGpVersion(gpHome string) (string, error) {
	pgGpVersionCmd := &postgres.Postgres{GpVersion: true}
	out, err := utils.RunGpCommand(pgGpVersionCmd, gpHome)
	if err != nil {
		return "", fmt.Errorf("fetching postgres gp-version: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

func GetDefaultHubLogDir() string {
	currentUser, _ := utils.System.CurrentUser()

	return filepath.Join(currentUser.HomeDir, "gpAdminLogs")
}

// GetCoordinatorConn creates a connection object for the coordinator segment
// given only its data directory. This function is expected to be called on the
// coordinator host only. By default it creates a non utility mode connection
// and uses the 'template1' database if no database is provided
func GetCoordinatorConn(datadir, dbname string, utility ...bool) (*dbconn.DBConn, error) {
	value, err := postgres.GetConfigValue(datadir, "port")
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}

	if dbname == "" {
		dbname = constants.DefaultDatabase
	}
	conn := newDBConnFromEnvironment(dbname)
	conn.Port = port

	err = conn.Connect(1, utility...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func TriggerFtsProbe(coordinatorDataDir string) error {
	conn, err := GetCoordinatorConn(coordinatorDataDir, "")
	if err != nil {
		return err
	}
	defer conn.Close()

	query := "SELECT gp_request_fts_probe_scan()"
	_, err = conn.Exec(query)
	gplog.Debug("Executing query %q", query)
	if err != nil {
		return fmt.Errorf("triggering FTS probe: %w", err)
	}

	return nil
}

// used only for testing
func SetNewDBConnFromEnvironment(customFunc func(dbname string) *dbconn.DBConn) {
	newDBConnFromEnvironment = customFunc
}

func ResetNewDBConnFromEnvironment() {
	newDBConnFromEnvironment = dbconn.NewDBConnFromEnvironment
}
