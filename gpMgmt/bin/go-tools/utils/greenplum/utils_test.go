package greenplum_test

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/testutils/exectest"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/greenplum"
)

func init() {
	exectest.RegisterMains(
		PgVersionCmd,
	)
}

func TestGetPostgresGpVersion(t *testing.T) {
	testhelper.SetupTestLogger()
	t.Run("returns error if command execution fails", func(t *testing.T) {
		expectedStr := "fetching postgres gp-version:"
		utils.System.ExecCommand = exectest.NewCommand(exectest.Failure)
		defer utils.ResetSystemFunctions()
		_, err := greenplum.GetPostgresGpVersion("/gpHome")
		if err == nil || !strings.Contains(err.Error(), expectedStr) {
			t.Fatalf("expected errror: `%s`, got error:`%v`", expectedStr, err)
		}

	})
	t.Run("returns no error when command is successful", func(t *testing.T) {
		utils.System.ExecCommand = exectest.NewCommand(exectest.Success)
		defer utils.ResetSystemFunctions()
		_, err := greenplum.GetPostgresGpVersion("/gpHome")
		if err != nil {
			t.Fatalf("expected no errror, got error:`%v`", err)
		}

	})
	t.Run("returns gp-version correctly by eliminating trailing spaces", func(t *testing.T) {
		expectedStr := "test-version-1234"
		utils.System.ExecCommand = exectest.NewCommand(PgVersionCmd)
		defer utils.ResetSystemFunctions()
		version, err := greenplum.GetPostgresGpVersion("/gpHome")
		if err != nil || !strings.Contains(version, expectedStr) {
			t.Fatalf("expected version: `%s`, got version:`%v`", expectedStr, version)
		}

	})
}

func TestGetDefaultHubLogDir(t *testing.T) {
	t.Run("correctly returns the default log directory", func(t *testing.T) {
		utils.System.CurrentUser = func() (*user.User, error) {
			return &user.User{
				HomeDir: "home",
			}, nil
		}
		defer utils.ResetSystemFunctions()

		result := greenplum.GetDefaultHubLogDir()
		expected := "home/gpAdminLogs"
		if result != expected {
			t.Fatalf("got %v, want %v", result, expected)
		}
	})
}

func TestGetCoordinatorConn(t *testing.T) {
	t.Run("successfully returns the coordinator connection", func(t *testing.T) {
		expectedPort := 1234
		expectedDatadir := "gpseg1"
		utils.System.Open = func(name string) (*os.File, error) {
			if filepath.Dir(name) != expectedDatadir {
				t.Fatalf("got %s, want %s", filepath.Dir(name), expectedDatadir)
			}

			reader, writer, _ := os.Pipe()
			_, err := writer.WriteString(fmt.Sprintf("port = %d", expectedPort))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			writer.Close()

			return reader, nil
		}
		defer utils.ResetSystemFunctions()

		greenplum.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			if dbname != constants.DefaultDatabase {
				t.Fatalf("got %s, want %s", dbname, constants.DefaultDatabase)
			}

			conn, mock := testutils.CreateMockDBConn(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")

			return conn
		})
		defer greenplum.ResetNewDBConnFromEnvironment()

		conn, err := greenplum.GetCoordinatorConn(expectedDatadir, "")
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		if conn.Port != 1234 {
			t.Fatalf("got %d, want %d", conn.Port, expectedPort)
		}
	})

	t.Run("successfully returns the coordinator connection in utility mode", func(t *testing.T) {
		expectedPort := 1234
		expectedDatadir := "gpseg1"
		utils.System.Open = func(name string) (*os.File, error) {
			if filepath.Dir(name) != expectedDatadir {
				t.Fatalf("got %s, want %s", filepath.Dir(name), expectedDatadir)
			}

			reader, writer, _ := os.Pipe()
			_, err := writer.WriteString(fmt.Sprintf("port = %d", expectedPort))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			writer.Close()

			return reader, nil
		}
		defer utils.ResetSystemFunctions()

		greenplum.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			if dbname != constants.DefaultDatabase {
				t.Fatalf("got %s, want %s", dbname, constants.DefaultDatabase)
			}

			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")

			return conn
		})
		defer greenplum.ResetNewDBConnFromEnvironment()

		conn, err := greenplum.GetCoordinatorConn(expectedDatadir, "", true)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		if conn.Port != 1234 {
			t.Fatalf("got %d, want %d", conn.Port, expectedPort)
		}
	})

	t.Run("errors out when not able to get the port value from the config", func(t *testing.T) {
		expectedErr := errors.New("error")
		utils.System.Open = func(name string) (*os.File, error) {
			return nil, expectedErr
		}

		_, err := greenplum.GetCoordinatorConn("gpseg1", "")
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		utils.System.Open = func(name string) (*os.File, error) {
			reader, writer, _ := os.Pipe()
			_, err := writer.WriteString("port = abc")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			writer.Close()

			return reader, nil
		}
		defer utils.ResetSystemFunctions()

		_, err = greenplum.GetCoordinatorConn("gpseg1", "")
		expectedErr = strconv.ErrSyntax
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})

	t.Run("errors out when not able to connect to the database", func(t *testing.T) {
		expectedErr := errors.New("error")
		utils.System.Open = func(name string) (*os.File, error) {
			reader, writer, _ := os.Pipe()
			_, err := writer.WriteString("port = 1234")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			writer.Close()

			return reader, nil
		}

		greenplum.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, _ := testutils.CreateMockDBConn(t, expectedErr)
			return conn
		})
		defer greenplum.ResetNewDBConnFromEnvironment()

		_, err := greenplum.GetCoordinatorConn("gpseg1", "")
		if !strings.HasPrefix(err.Error(), expectedErr.Error()) {
			t.Fatalf("got %v, want prefix %s", err, expectedErr)
		}
	})
}

func PgVersionCmd() {
	os.Stdout.WriteString("   test-version-1234   ")
}
