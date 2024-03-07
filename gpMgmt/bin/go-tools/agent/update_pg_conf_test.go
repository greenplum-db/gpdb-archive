package agent_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/agent"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
)

func TestUpdatePgConf(t *testing.T) {
	testhelper.SetupTestLogger()

	agentServer := agent.New(agent.Config{
		GpHome: "gpHome",
	})

	cases := []struct {
		request  *idl.UpdatePgConfRequest
		expected string
	}{
		{
			request: &idl.UpdatePgConfRequest{
				Pgdata: "gpseg",
				Params: map[string]string{
					"guc1": "value1",
					"guc2": "value2",
				},
				Overwrite: false,
			},
			expected: `guc1 = 'value1' # guc1 = old_value1
guc2 = 'value2'`,
		},
		{
			request: &idl.UpdatePgConfRequest{
				Pgdata: "gpseg",
				Params: map[string]string{
					"guc1": "value1",
					"guc2": "value2",
				},
				Overwrite: true,
			},
			expected: `guc1 = 'value1'
guc2 = 'value2'`,
		},
	}

	for _, tc := range cases {
		t.Run("successfully updates the segment postgresql.conf", func(t *testing.T) {
			var reader, writer *os.File
			utils.System.Open = func(name string) (*os.File, error) {
				if !strings.HasPrefix(name, tc.request.Pgdata) {
					t.Fatalf("got %s, want prefix %s", name, tc.request.Pgdata)
				}

				reader, writer, _ := os.Pipe()
				_, _ = writer.WriteString("guc1 = old_value1")
				writer.Close()

				return reader, nil
			}
			utils.System.Create = func(name string) (*os.File, error) {
				reader, writer, _ = os.Pipe()

				return writer, nil
			}
			defer utils.ResetSystemFunctions()

			_, err := agentServer.UpdatePgConf(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
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

	t.Run("returns error when not able to update the postgresql.conf file", func(t *testing.T) {
		expectedErr := errors.New("error")
		utils.System.Open = func(name string) (*os.File, error) {
			return nil, expectedErr
		}
		defer utils.ResetSystemFunctions()

		_, err := agentServer.UpdatePgConf(context.Background(), &idl.UpdatePgConfRequest{})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		expectedErrPrefix := "updating postgresql.conf"
		if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
			t.Fatalf("got %v, want prefix %s", err, expectedErrPrefix)
		}
	})
}
