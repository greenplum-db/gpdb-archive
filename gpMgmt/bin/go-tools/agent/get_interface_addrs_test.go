package agent_test

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/greenplum-db/gpdb/gp/agent"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/utils"
)

func TestGetInterfaceAddrs(t *testing.T) {
	agentServer := agent.New(agent.Config{
		GpHome: "gpHome",
	})

	t.Run("succesfully returns the interface addresses without the loopback", func(t *testing.T) {
		expected := []string{"192.0.1.0/24", "2001:db8::/32"}

		utils.System.InterfaceAddrs = func() ([]net.Addr, error) {
			_, addr1, _ := net.ParseCIDR(expected[0])
			_, addr2, _ := net.ParseCIDR(expected[1])
			_, loopbackAddrIp4, _ := net.ParseCIDR("127.0.0.1/8")
			_, loopbackAddrIp6, _ := net.ParseCIDR("::1/128")

			return []net.Addr{addr1, addr2, loopbackAddrIp4, loopbackAddrIp6}, nil
		}
		defer utils.ResetSystemFunctions()

		resp, err := agentServer.GetInterfaceAddrs(context.Background(), &idl.GetInterfaceAddrsRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(resp.Addrs, expected) {
			t.Fatalf("got %+v, want %+v", resp.Addrs, expected)
		}
	})

	t.Run("errors out when not able to get the host addreses", func(t *testing.T) {
		expectedErr := errors.New("error")
		utils.System.InterfaceAddrs = func() ([]net.Addr, error) {

			return nil, expectedErr
		}
		defer utils.ResetSystemFunctions()

		_, err := agentServer.GetInterfaceAddrs(context.Background(), &idl.GetInterfaceAddrsRequest{})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}
