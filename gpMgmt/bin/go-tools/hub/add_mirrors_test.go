package hub_test

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/hub"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/idl/mock_idl"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/greenplum"
)

var (
	gparray     *greenplum.GpArray
	coordinator *greenplum.Segment
	primary1    *greenplum.Segment
	primary2    *greenplum.Segment
	mirror1     *greenplum.Segment
	mirror2     *greenplum.Segment
	hubServer   *hub.Server
)

func initialize(t *testing.T) {
	coordinator = createSegment(t, 1, -1, constants.RolePrimary, constants.RolePrimary, 7000, "cdw", "cdw", "/data/primary/gpseg-1")
	primary1 = createSegment(t, 2, 0, constants.RolePrimary, constants.RolePrimary, 7001, "sdw1", "sdw1", "/data/primary/gpseg0")
	primary2 = createSegment(t, 4, 1, constants.RolePrimary, constants.RolePrimary, 7003, "sdw2", "sdw2", "/data/primary/gpseg1")
	mirror1 = createSegment(t, 3, 0, constants.RoleMirror, constants.RoleMirror, 7002, "sdw2", "sdw2", "/data/mirror/gpseg0")
	mirror2 = createSegment(t, 5, 1, constants.RoleMirror, constants.RoleMirror, 7004, "sdw1", "sdw1", "/data/mirror/gpseg1")

	gparray = &greenplum.GpArray{
		Coordinator: coordinator,
		SegmentPairs: []greenplum.SegmentPair{
			{
				Primary: primary1,
				Mirror:  mirror1,
			},
			{
				Primary: primary2,
				Mirror:  mirror2,
			},
		},
	}

	credentials := &testutils.MockCredentials{TlsConnection: insecure.NewCredentials()}
	hubConfig := &hub.Config{
		1234,
		5678,
		[]string{"sdw1", "sdw2"},
		"/tmp/logDir",
		"gp",
		"gpHome",
		credentials,
	}
	hubServer = hub.New(hubConfig, nil)
}

func TestCreateMirrorSegments(t *testing.T) {
	_, _, logfile := testhelper.SetupTestLogger()
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

	t.Run("successfully creates the mirror segments", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().PgBasebackup(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.PgBasebackupResponse{}, nil)
		sdw1.EXPECT().UpdatePgConf(
			gomock.Any(),
			&idl.UpdatePgConfRequest{
				Pgdata: mirror2.DataDir,
				Params: map[string]string{
					"port": strconv.Itoa(mirror2.Port),
				},
				Overwrite: true,
			},
		).Return(&idl.UpdatePgConfRespoonse{}, nil)

		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().PgBasebackup(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.PgBasebackupResponse{}, nil)
		sdw2.EXPECT().UpdatePgConf(
			gomock.Any(),
			&idl.UpdatePgConfRequest{
				Pgdata: mirror1.DataDir,
				Params: map[string]string{
					"port": strconv.Itoa(mirror1.Port),
				},
				Overwrite: true,
			},
		).Return(&idl.UpdatePgConfRespoonse{}, nil)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		mock, stream := testutils.NewMockStream()
		err := hubServer.CreateMirrorSegments(mock, gparray, mirrorSegs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedStreamResponse := make([]*idl.HubReply, len(mirrorSegs)+1)
		for i := range expectedStreamResponse {
			expectedStreamResponse[i] = &idl.HubReply{
				Message: &idl.HubReply_ProgressMsg{
					ProgressMsg: &idl.ProgressMessage{
						Label: "Initializing mirror segments:",
						Total: int32(len(mirrorSegs)),
					},
				},
			}
		}

		if !reflect.DeepEqual(stream.GetBuffer(), expectedStreamResponse) {
			t.Fatalf("got %+v, want %+v", stream.GetBuffer(), expectedStreamResponse)
		}

		testutils.AssertLogMessage(t, logfile, `\[DEBUG\]:-Starting to create mirror segment`)
		testutils.AssertLogMessage(t, logfile, `\[DEBUG\]:-Successfully ran pg_basebackup`)
		testutils.AssertLogMessage(t, logfile, `\[DEBUG\]:-Starting to modify the postgresql.conf`)
		testutils.AssertLogMessage(t, logfile, `\[DEBUG\]:-Successfully modified the postgresql.conf`)
		testutils.AssertLogMessage(t, logfile, `\[DEBUG\]:-Successfully created mirror segment`)
	})

	t.Run("errors out when fails to run pg_basebackup", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("error")
		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().PgBasebackup(
			gomock.Any(),
			gomock.Any(),
		).Return(nil, expectedErr)

		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().PgBasebackup(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.PgBasebackupResponse{}, nil)
		sdw2.EXPECT().UpdatePgConf(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.UpdatePgConfRespoonse{}, nil)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		mock, stream := testutils.NewMockStream()
		err := hubServer.CreateMirrorSegments(mock, gparray, mirrorSegs)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		expectedErrString := fmt.Sprintf("host: sdw1, %v", expectedErr)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}

		expectedStreamResponse := make([]*idl.HubReply, len(mirrorSegs))
		for i := range expectedStreamResponse {
			expectedStreamResponse[i] = &idl.HubReply{
				Message: &idl.HubReply_ProgressMsg{
					ProgressMsg: &idl.ProgressMessage{
						Label: "Initializing mirror segments:",
						Total: int32(len(mirrorSegs)),
					},
				},
			}
		}

		if !reflect.DeepEqual(stream.GetBuffer(), expectedStreamResponse) {
			t.Fatalf("got %+v, want %+v", stream.GetBuffer(), expectedStreamResponse)
		}
	})

	t.Run("errors out when fails to update the postgresql.conf", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("error")
		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().PgBasebackup(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.PgBasebackupResponse{}, nil)
		sdw1.EXPECT().UpdatePgConf(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.UpdatePgConfRespoonse{}, nil)

		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().PgBasebackup(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.PgBasebackupResponse{}, nil)
		sdw2.EXPECT().UpdatePgConf(
			gomock.Any(),
			gomock.Any(),
		).Return(nil, expectedErr)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		mock, stream := testutils.NewMockStream()
		err := hubServer.CreateMirrorSegments(mock, gparray, mirrorSegs)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		expectedErrString := fmt.Sprintf("host: sdw2, %v", expectedErr)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}

		expectedStreamResponse := make([]*idl.HubReply, len(mirrorSegs))
		for i := range expectedStreamResponse {
			expectedStreamResponse[i] = &idl.HubReply{
				Message: &idl.HubReply_ProgressMsg{
					ProgressMsg: &idl.ProgressMessage{
						Label: "Initializing mirror segments:",
						Total: int32(len(mirrorSegs)),
					},
				},
			}
		}

		if !reflect.DeepEqual(stream.GetBuffer(), expectedStreamResponse) {
			t.Fatalf("got %+v, want %+v", stream.GetBuffer(), expectedStreamResponse)
		}
	})

	t.Run("errors out when not able to find the mirror content in gparray", func(t *testing.T) {
		segs := []*idl.Segment{{Contentid: 1234}}

		mock, stream := testutils.NewMockStream()
		err := hubServer.CreateMirrorSegments(mock, gparray, segs)

		expectedErrString := "could not find any segments with content 1234"
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}

		if len(stream.GetBuffer()) != 0 {
			t.Fatalf("expected no stream responses, got %+v", stream.GetBuffer())
		}
	})
}

func TestStartMirrorSegments(t *testing.T) {
	testhelper.SetupTestLogger()
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

	t.Run("successfully starts the mirror segments", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().StartSegment(
			gomock.Any(),
			&idl.StartSegmentRequest{
				DataDir: mirrorSegs[1].DataDirectory,
				Wait:    true,
				Options: "-c gp_role=execute",
			},
		).Return(&idl.StartSegmentReply{}, nil)

		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().StartSegment(
			gomock.Any(),
			&idl.StartSegmentRequest{
				DataDir: mirrorSegs[0].DataDirectory,
				Wait:    true,
				Options: "-c gp_role=execute",
			},
		).Return(&idl.StartSegmentReply{}, nil)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		err := hubServer.StartMirrorSegments(mirrorSegs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("errors out when not able to start the segment", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		sdw1 := mock_idl.NewMockAgentClient(ctrl)
		sdw1.EXPECT().StartSegment(
			gomock.Any(),
			gomock.Any(),
		).Return(&idl.StartSegmentReply{}, nil)

		expectedErr := errors.New("error")
		sdw2 := mock_idl.NewMockAgentClient(ctrl)
		sdw2.EXPECT().StartSegment(
			gomock.Any(),
			gomock.Any(),
		).Return(nil, expectedErr)

		agentConns := []*hub.Connection{
			{AgentClient: sdw1, Hostname: "sdw1"},
			{AgentClient: sdw2, Hostname: "sdw2"},
		}
		hubServer.Conns = agentConns

		err := hubServer.StartMirrorSegments(mirrorSegs)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got%#v, want %#v", err, expectedErr)
		}

		expectedErrString := fmt.Sprintf("host: sdw2, %v", expectedErr)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})
}

func TestAddMirrors(t *testing.T) {
	testhelper.SetupTestLogger()

	testhelper.SetupTestLogger()
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

	t.Run("successfully adds the mirror segments to the cluster", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hub.SetEnsureConnectionsAreReady(func(conns []*hub.Connection) error {
			return nil
		})
		defer hub.ResetEnsureConnectionsAreReady()

		utils.System.Open = func(name string) (*os.File, error) {
			reader, writer, _ := os.Pipe()
			defer writer.Close()

			_, err := writer.WriteString("port=1234")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			return reader, nil
		}
		defer utils.ResetSystemFunctions()

		var called bool
		greenplum.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			var conn *dbconn.DBConn
			var mock sqlmock.Sqlmock

			if !called {
				called = true
				conn, mock = testutils.CreateMockDBConnForUtilityMode(t)
				testhelper.ExpectVersionQuery(mock, "7.0.0")

				rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
				addSegmentRows(t, rows, coordinator, primary1, primary2)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
				mock.ExpectExec("SELECT").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("SELECT").WillReturnResult(sqlmock.NewResult(1, 1))
				rows = sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
				addSegmentRows(t, rows, coordinator, primary1, primary2, mirror1, mirror2)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			} else {
				conn, mock = testutils.CreateMockDBConn(t)
				testhelper.ExpectVersionQuery(mock, "7.0.0")
				mock.ExpectExec("SELECT").WillReturnResult(sqlmock.NewResult(1, 1))
			}

			return conn
		})
		defer greenplum.ResetNewDBConnFromEnvironment()

		hubServer.Conns = createMockClients(t, ctrl, ErrorType{})

		_, stream := testutils.NewMockStream()
		err := hubServer.AddMirrors(&idl.AddMirrorsRequest{HbaHostnames: true, Mirrors: mirrorSegs}, stream)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("when cluster already has mirrors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hub.SetEnsureConnectionsAreReady(func(conns []*hub.Connection) error {
			return nil
		})
		defer hub.ResetEnsureConnectionsAreReady()

		utils.System.Open = func(name string) (*os.File, error) {
			reader, writer, _ := os.Pipe()
			defer writer.Close()

			_, err := writer.WriteString("port=1234")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			return reader, nil
		}
		defer utils.ResetSystemFunctions()

		greenplum.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")

			rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
			addSegmentRows(t, rows, coordinator, primary1, primary2, mirror1, mirror2)
			mock.ExpectQuery("SELECT").WillReturnRows(rows)
			return conn
		})
		defer greenplum.ResetNewDBConnFromEnvironment()

		hubServer.Conns = createMockClients(t, ctrl, ErrorType{})

		_, stream := testutils.NewMockStream()
		err := hubServer.AddMirrors(&idl.AddMirrorsRequest{HbaHostnames: true, Mirrors: mirrorSegs}, stream)
		expectedErrString := "cannot add mirrors, the cluster is already configured with mirrors"
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("when number of mirror segments are not equal to the primary segments", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hub.SetEnsureConnectionsAreReady(func(conns []*hub.Connection) error {
			return nil
		})
		defer hub.ResetEnsureConnectionsAreReady()

		utils.System.Open = func(name string) (*os.File, error) {
			reader, writer, _ := os.Pipe()
			defer writer.Close()

			_, err := writer.WriteString("port=1234")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			return reader, nil
		}
		defer utils.ResetSystemFunctions()

		greenplum.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")

			rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
			addSegmentRows(t, rows, coordinator, primary1)
			mock.ExpectQuery("SELECT").WillReturnRows(rows)
			return conn
		})
		defer greenplum.ResetNewDBConnFromEnvironment()

		hubServer.Conns = createMockClients(t, ctrl, ErrorType{})

		_, stream := testutils.NewMockStream()
		err := hubServer.AddMirrors(&idl.AddMirrorsRequest{HbaHostnames: true, Mirrors: mirrorSegs}, stream)
		expectedErrString := "number of mirrors 2 is not equal to the number of primaries 1 present in the cluster"
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	expectedErr := errors.New("error")
	cases := []ErrorType{
		{
			PgBasebackup: expectedErr,
		},
		{
			UpdatePgConf: expectedErr,
		},
		{
			UpdatePgHbaConf: expectedErr,
		},
		{
			StartSegment: expectedErr,
		},
	}
	for _, tc := range cases {
		t.Run("returns appropriate error during different RPC calls", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			hub.SetEnsureConnectionsAreReady(func(conns []*hub.Connection) error {
				return nil
			})
			defer hub.ResetEnsureConnectionsAreReady()

			utils.System.Open = func(name string) (*os.File, error) {
				reader, writer, _ := os.Pipe()
				defer writer.Close()

				_, err := writer.WriteString("port=1234")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				return reader, nil
			}
			defer utils.ResetSystemFunctions()

			var called bool
			greenplum.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
				var conn *dbconn.DBConn
				var mock sqlmock.Sqlmock

				if !called {
					called = true
					conn, mock = testutils.CreateMockDBConnForUtilityMode(t)
					testhelper.ExpectVersionQuery(mock, "7.0.0")

					rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
					addSegmentRows(t, rows, coordinator, primary1, primary2)
					mock.ExpectQuery("SELECT").WillReturnRows(rows)
					mock.ExpectExec("SELECT").WillReturnResult(sqlmock.NewResult(1, 1))
					mock.ExpectExec("SELECT").WillReturnResult(sqlmock.NewResult(1, 1))
					rows = sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
					addSegmentRows(t, rows, coordinator, primary1, primary2, mirror1, mirror2)
					mock.ExpectQuery("SELECT").WillReturnRows(rows)
				} else {
					conn, mock = testutils.CreateMockDBConn(t)
					testhelper.ExpectVersionQuery(mock, "7.0.0")
					mock.ExpectExec("SELECT").WillReturnResult(sqlmock.NewResult(1, 1))
				}

				return conn
			})
			defer greenplum.ResetNewDBConnFromEnvironment()

			hubServer.Conns = createMockClients(t, ctrl, tc)

			_, stream := testutils.NewMockStream()
			err := hubServer.AddMirrors(&idl.AddMirrorsRequest{HbaHostnames: true, Mirrors: mirrorSegs}, stream)
			if !errors.Is(err, expectedErr) {
				t.Fatalf("got %#v, want %#v", err, expectedErr)
			}
		})
	}
}

func createSegment(t *testing.T, dbid int, content int, role string, preferredRole string, port int, hostname string, address string, dataDir string) *greenplum.Segment {
	t.Helper()

	return &greenplum.Segment{
		Dbid:          dbid,
		Content:       content,
		Role:          role,
		PreferredRole: preferredRole,
		Port:          port,
		Hostname:      hostname,
		Address:       address,
		DataDir:       dataDir,
	}
}

type ErrorType struct {
	PgBasebackup    error
	UpdatePgConf    error
	StartSegment    error
	UpdatePgHbaConf error
}

func createMockClients(t *testing.T, ctrl *gomock.Controller, errorType ErrorType) []*hub.Connection {
	t.Helper()

	cdw := mock_idl.NewMockAgentClient(ctrl)
	sdw1 := mock_idl.NewMockAgentClient(ctrl)
	sdw2 := mock_idl.NewMockAgentClient(ctrl)

	sdw1.EXPECT().PgBasebackup(gomock.Any(), gomock.Any()).Return(nil, errorType.PgBasebackup).AnyTimes()
	sdw1.EXPECT().UpdatePgConf(gomock.Any(), gomock.Any()).Return(nil, errorType.UpdatePgConf).AnyTimes()
	sdw1.EXPECT().StartSegment(gomock.Any(), gomock.Any()).Return(nil, errorType.StartSegment).AnyTimes()
	sdw1.EXPECT().UpdatePgHbaConfAndReload(gomock.Any(), gomock.Any()).Return(nil, errorType.UpdatePgHbaConf).AnyTimes()

	sdw2.EXPECT().PgBasebackup(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	sdw2.EXPECT().UpdatePgConf(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	sdw2.EXPECT().StartSegment(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	sdw2.EXPECT().UpdatePgHbaConfAndReload(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	return []*hub.Connection{
		{AgentClient: cdw, Hostname: "cdw"},
		{AgentClient: sdw1, Hostname: "sdw1"},
		{AgentClient: sdw2, Hostname: "sdw2"},
	}
}

func addSegmentRows(t *testing.T, rows *sqlmock.Rows, segs ...*greenplum.Segment) {
	t.Helper()

	for _, seg := range segs {
		rows.AddRow(seg.Dbid, seg.Content, seg.Role, seg.PreferredRole, seg.Port, seg.Hostname, seg.Address, seg.DataDir)
	}
}
