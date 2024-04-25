package hub_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/hub"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestServer_GetAllHostNames(t *testing.T) {
	testhelper.SetupTestLogger()

	credentials := &testutils.MockCredentials{TlsConnection: insecure.NewCredentials()}
	listener := bufconn.Listen(1024 * 1024)
	hubConfig := &hub.Config{
		1234,
		5678,
		[]string{"sdw1", "sdw2"},
		"/tmp/logDir",
		"gp",
		"gpHome",
		credentials,
	}

	t.Run("returns error when fails to load client credentials", func(t *testing.T) {
		testStr := "test error"
		dialer := func(ctx context.Context, address string) (net.Conn, error) {
			if strings.HasPrefix(address, "sdw1") {
				return nil, errors.New(testStr)
			}
			return listener.Dial()
		}
		hubServer := hub.New(hubConfig, dialer)
		request := idl.GetAllHostNamesRequest{HostList: []string{"sdw1", "sdw2"}}

		_, err := hubServer.GetAllHostNames(context.Background(), &request)
		if err == nil || !strings.Contains(err.Error(), testStr) {
			t.Fatalf("Got:%v, expected:%s", err, testStr)
		}
	})
}
