package hub_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/greenplum-db/gpdb/gp/hub"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/idl/mock_idl"
)

func TestUpdatePgHbaConf(t *testing.T) {
	initialize(t)

	mirrorSegs := []*idl.Segment{
		{
			Port:          int32(mirror1.Port),
			HostName:      mirror1.Hostname,
			HostAddress:   mirror1.Address,
			DataDirectory: mirror1.DataDir,
			Contentid:     int32(mirror1.Content),
		},
		{
			Port:          int32(mirror2.Port),
			HostName:      mirror2.Hostname,
			HostAddress:   mirror2.Address,
			DataDirectory: mirror2.DataDir,
			Contentid:     int32(mirror2.Content),
		},
	}

	t.Run("succesfully updates the pg_hba.conf file when hba_hostnames is false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		addrMap := map[string][]string{
			"sdw1": {"192.0.1.0/24"},
			"sdw2": {"192.0.2.0/24"},
		}

		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().UpdatePgHbaConfAndReload(
			gomock.Any(),
			&idl.UpdatePgHbaConfRequest{
				Pgdata:      primary1.DataDir,
				Addrs:       append(addrMap[primary1.Hostname], addrMap[mirror1.Hostname]...),
				Replication: true,
			},
		).Return(&idl.UpdatePgHbaConfResponse{}, nil)
		sdw1.EXPECT().GetInterfaceAddrs(
			gomock.Any(),
			&idl.GetInterfaceAddrsRequest{},
		).Return(&idl.GetInterfaceAddrsResponse{
			Addrs: addrMap[primary1.Hostname],
		}, nil).Times(2)

		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().UpdatePgHbaConfAndReload(
			gomock.Any(),
			&idl.UpdatePgHbaConfRequest{
				Pgdata:      primary2.DataDir,
				Addrs:       append(addrMap[primary2.Hostname], addrMap[mirror2.Hostname]...),
				Replication: true,
			},
		).Return(&idl.UpdatePgHbaConfResponse{}, nil)
		sdw2.EXPECT().GetInterfaceAddrs(
			gomock.Any(),
			&idl.GetInterfaceAddrsRequest{},
		).Return(&idl.GetInterfaceAddrsResponse{
			Addrs: addrMap[primary2.Hostname],
		}, nil).Times(2)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		err := hubServer.UpdatePgHbaConfWithMirrorEntries(gparray, mirrorSegs, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("succesfully updates the pg_hba.conf file when hba_hostnames is true", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().UpdatePgHbaConfAndReload(
			gomock.Any(),
			&idl.UpdatePgHbaConfRequest{
				Pgdata:      primary1.DataDir,
				Addrs:       []string{primary1.Hostname, mirror1.Hostname},
				Replication: true,
			},
		).Return(&idl.UpdatePgHbaConfResponse{}, nil)

		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().UpdatePgHbaConfAndReload(
			gomock.Any(),
			&idl.UpdatePgHbaConfRequest{
				Pgdata:      primary2.DataDir,
				Addrs:       []string{primary2.Hostname, mirror2.Hostname},
				Replication: true,
			},
		).Return(&idl.UpdatePgHbaConfResponse{}, nil)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		err := hubServer.UpdatePgHbaConfWithMirrorEntries(gparray, mirrorSegs, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("errors out when not able to find the mirror content in gparray", func(t *testing.T) {
		segs := []*idl.Segment{{Contentid: 1234}}
		err := hubServer.UpdatePgHbaConfWithMirrorEntries(gparray, segs, true)

		expectedErrString := "could not find any segments with content 1234"
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("errors out when not able to get the interface addresses for a host", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		addrMap := map[string][]string{
			"sdw1": {"192.0.1.0/24"},
			"sdw2": {"192.0.2.0/24"},
		}
		expectedErr := errors.New("error")

		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().GetInterfaceAddrs(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.GetInterfaceAddrsResponse{
			Addrs: addrMap["sdw1"],
		}, nil).Times(2)

		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().UpdatePgHbaConfAndReload(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.UpdatePgHbaConfResponse{}, nil)
		sdw2.EXPECT().GetInterfaceAddrs(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.GetInterfaceAddrsResponse{
			Addrs: addrMap["sdw2"],
		}, nil)
		sdw2.EXPECT().GetInterfaceAddrs(
			gomock.Any(),
			gomock.Any(),
		).Return(nil, expectedErr)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		err := hubServer.UpdatePgHbaConfWithMirrorEntries(gparray, mirrorSegs, false)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		expectedErrString := "failed to get interface addresses for host sdw2"
		if !strings.Contains(err.Error(), expectedErrString) {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("errors out when fails to modify the pg_hba.conf", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("error")

		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().UpdatePgHbaConfAndReload(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.UpdatePgHbaConfResponse{}, nil)

		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().UpdatePgHbaConfAndReload(
			gomock.Any(),
			gomock.Any(),
		).Return(nil, expectedErr)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		err := hubServer.UpdatePgHbaConfWithMirrorEntries(gparray, mirrorSegs, true)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		expectedErrString := fmt.Sprintf("host: sdw2, %v", expectedErr)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})
}
