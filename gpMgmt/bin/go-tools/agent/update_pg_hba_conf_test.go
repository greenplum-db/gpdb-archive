package agent_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/agent"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/testutils/exectest"
	"github.com/greenplum-db/gpdb/gp/utils"
)

func TestUpdatePgHbaConf(t *testing.T) {
	testhelper.SetupTestLogger()

	agentServer := agent.New(agent.Config{
		GpHome: "gpHome",
	})

	cases := []struct {
		request  *idl.UpdatePgHbaConfRequest
		expected string
	}{
		{
			request: &idl.UpdatePgHbaConfRequest{
				Pgdata:      "gpseg",
				Addrs:       []string{"sdw1", "sdw2"},
				Replication: true,
			},
			expected: `
host	all	gpadmin	sdw1	trust
host	all	gpadmin	sdw2	trust
host	replication	gpadmin	samehost	trust
host	replication	gpadmin	sdw1	trust
host	replication	gpadmin	sdw2	trust`,
		},
		{
			request: &idl.UpdatePgHbaConfRequest{
				Pgdata:      "gpseg",
				Addrs:       []string{"sdw1", "sdw2"},
				Replication: false,
			},
			expected: `
host	all	gpadmin	sdw1	trust
host	all	gpadmin	sdw2	trust`,
		},
	}

	for _, tc := range cases {
		t.Run("successfully updates the segment pg_hba.conf", func(t *testing.T) {
			utils.System.CurrentUser = func() (*user.User, error) {
				return &user.User{Username: "gpadmin"}, nil
			}

			var pgCtlCalled bool
			utils.System.ExecCommand = exectest.NewCommandWithVerifier(exectest.Success, func(utililty string, args ...string) {
				pgCtlCalled = true

				expectedUtility := "pg_ctl"
				if !strings.HasSuffix(utililty, expectedUtility) {
					t.Fatalf("got %s, want %s", utililty, expectedUtility)
				}

				expectedArgs := []string{"reload", "--pgdata", tc.request.Pgdata}
				if !reflect.DeepEqual(args, expectedArgs) {
					t.Fatalf("got %+v, want %+v", args, expectedArgs)
				}
			})

			var reader, writer *os.File
			utils.System.ReadFile = func(name string) ([]byte, error) {
				if !strings.HasPrefix(name, tc.request.Pgdata) {
					t.Fatalf("got %s, want prefix %s", name, tc.request.Pgdata)
				}
				return nil, nil
			}
			utils.System.Create = func(name string) (*os.File, error) {
				reader, writer, _ = os.Pipe()

				return writer, nil
			}
			defer utils.ResetSystemFunctions()

			_, err := agentServer.UpdatePgHbaConfAndReload(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !pgCtlCalled {
				t.Fatalf("expected pg_ctl to be called")
			}

			var buf = make([]byte, 1024)
			n, err := reader.Read(buf)
			if err != nil {
				t.Fatalf(err.Error())
			}
			result := string(buf[:n])

			if result != tc.expected {
				t.Fatalf("got %s, want %s", result, tc.expected)
			}
		})
	}

	t.Run("returns error when not able to update the pg_hba.conf file", func(t *testing.T) {
		expectedErr := errors.New("error")
		utils.System.ReadFile = func(name string) ([]byte, error) {
			return nil, expectedErr
		}
		defer utils.ResetSystemFunctions()

		_, err := agentServer.UpdatePgHbaConfAndReload(context.Background(), &idl.UpdatePgHbaConfRequest{})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		expectedErrPrefix := "updating pg_hba.conf"
		if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
			t.Fatalf("got %v, want prefix %s", err, expectedErrPrefix)
		}
	})

	t.Run("returns error when not able to pg_ctl reload", func(t *testing.T) {
		utils.System.ExecCommand = exectest.NewCommand(exectest.Failure)
		utils.System.ReadFile = func(name string) ([]byte, error) {
			return []byte{}, nil
		}
		utils.System.Create = func(name string) (*os.File, error) {
			_, writer, _ := os.Pipe()

			return writer, nil
		}
		defer utils.ResetSystemFunctions()

		_, err := agentServer.UpdatePgHbaConfAndReload(context.Background(), &idl.UpdatePgHbaConfRequest{})
		var expectedErr *exec.ExitError
		if !errors.As(err, &expectedErr) {
			t.Errorf("got %T, want %T", err, expectedErr)
		}

		expectedErrPrefix := "executing pg_ctl reload:"
		if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
			t.Fatalf("got %v, want prefix %s", err, expectedErrPrefix)
		}
	})
}
