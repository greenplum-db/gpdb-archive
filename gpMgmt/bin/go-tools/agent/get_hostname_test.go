package agent_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/agent"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
)

func TestGetHostName(t *testing.T) {
	testhelper.SetupTestLogger()
	agentServer := agent.New(agent.Config{
		GpHome: "gpHome",
	})

	t.Run("returns error when error getting hostname", func(t *testing.T) {
		testStr := "test error"
		defer utils.ResetSystemFunctions()
		utils.System.GetHostName = func() (name string, err error) {
			return "", fmt.Errorf(testStr)
		}

		_, err := agentServer.GetHostName(context.Background(), &idl.GetHostNameRequest{})
		if err == nil || !strings.Contains(err.Error(), testStr) {
			t.Fatalf("Got:%v, Expected:%s", err, testStr)
		}
	})

	t.Run("returns correct hostname", func(t *testing.T) {
		testStr := "test_hostname"
		defer utils.ResetSystemFunctions()
		utils.System.GetHostName = func() (name string, err error) {
			return testStr, nil
		}

		reply, err := agentServer.GetHostName(context.Background(), &idl.GetHostNameRequest{})
		if err != nil {
			t.Fatalf("Got:%v, Expected:no error", err)
		}
		if reply.Hostname != testStr {
			t.Fatalf("Got:%s, Expected:%s", reply.Hostname, testStr)
		}
	})
}
