package agent_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/agent"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/testutils/exectest"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/postgres"
)

func init() {
	exectest.RegisterMains()
}

func TestPgBasebackup(t *testing.T) {
	testhelper.SetupTestLogger()
	tempDir, _ := os.MkdirTemp("", "")

	agentServer := agent.New(agent.Config{
		GpHome: "gpHome",
		LogDir: tempDir,
	})

	request := &idl.PgBasebackupRequest{
		TargetDir:           "/mirror/gpseg0",
		SourceHost:          "sdw1",
		SourcePort:          1234,
		TargetDbid:          1,
		CreateSlot:          true,
		ReplicationSlotName: "test_slot",
	}

	t.Run("succesfully runs pg_basebackup when create slot is true", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		var pgBasebackupCalled bool
		utils.System.ExecCommand = exectest.NewCommandWithVerifier(exectest.Success, func(utility string, args ...string) {
			pgBasebackupCalled = true
			expectedUtility := "gpHome/bin/pg_basebackup"
			if utility != expectedUtility {
				t.Fatalf("got %s, want %s", utility, expectedUtility)
			}

			expectedArgs := []string{"--checkpoint", "fast", "--no-verify-checksums", "--progress", "--verbose", "--pgdata", request.TargetDir, "--host", request.SourceHost,
				"--port", strconv.Itoa(int(request.SourcePort)), "--create-slot", "--target-gp-dbid", strconv.Itoa(int(request.TargetDbid)), "--slot", request.ReplicationSlotName,
				"--wal-method", "stream", "--exclude", "./db_dumps", "--exclude", "./promote", "--exclude", "./db_analyze"}
			if !reflect.DeepEqual(args, expectedArgs) {
				t.Fatalf("got %+v, want %+v", args, expectedArgs)
			}
		})
		defer utils.ResetSystemFunctions()

		_, err := agentServer.PgBasebackup(context.Background(), request)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !pgBasebackupCalled {
			t.Fatalf("expected pg_basebackup to be called")
		}

		files, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, file := range files {
			if strings.HasPrefix(file.Name(), "pg_basebackup") {
				t.Fatalf("expected pg_basebackup files to be deleted")
			}
		}
	})

	t.Run("errors out when it fails to drop the replication slot", func(t *testing.T) {
		expectedErr := errors.New("error")
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnError(expectedErr)

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		var pgBasebackupCalled bool
		utils.System.ExecCommand = exectest.NewCommandWithVerifier(exectest.Success, func(utility string, args ...string) {
			pgBasebackupCalled = true
		})
		defer utils.ResetSystemFunctions()

		_, err := agentServer.PgBasebackup(context.Background(), request)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		expectedErrPrefix := fmt.Sprintf("failed to drop replication slot %s", request.ReplicationSlotName)
		if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
			t.Fatalf("got %v, want %s", err, expectedErrPrefix)
		}

		if pgBasebackupCalled {
			t.Fatalf("expected pg_basebackup to not be called")
		}
	})

	t.Run("errors out when fails to execute pg_basebackup", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		utils.System.ExecCommand = exectest.NewCommand(exectest.Failure)
		defer utils.ResetSystemFunctions()

		_, err := agentServer.PgBasebackup(context.Background(), request)
		var expectedErr *exec.ExitError
		if !errors.As(err, &expectedErr) {
			t.Errorf("got %T, want %T", err, expectedErr)
		}

		expectedErrPrefix := "executing pg_basebackup:"
		if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
			t.Fatalf("got %v, want %s", err, expectedErrPrefix)
		}

		var found bool
		files, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, file := range files {
			if strings.HasPrefix(file.Name(), "pg_basebackup") {
				found = true
			}
		}

		if !found {
			t.Fatalf("expected pg_basebackup files to not be deleted")
		}
	})
}
