package greenplum_test

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/utils/greenplum"
)

var (
	gparray     greenplum.GpArray
	coordinator *greenplum.Segment
	standby     *greenplum.Segment
	primary1    *greenplum.Segment
	primary2    *greenplum.Segment
	mirror1     *greenplum.Segment
	mirror2     *greenplum.Segment
)

func initializeGpArray(t *testing.T) {
	coordinator = createSegment(t, 1, -1, constants.RolePrimary, constants.RolePrimary, 7000, "cdw", "cdw", "/data/primary/gpseg-1")
	standby = createSegment(t, 2, -1, constants.RoleMirror, constants.RoleMirror, 7001, "scdw", "scdw", "/data/mirror/gpseg-1")
	primary1 = createSegment(t, 3, 0, constants.RolePrimary, constants.RolePrimary, 7002, "sdw1", "sdw1", "/data/primary/gpseg0")
	primary2 = createSegment(t, 4, 1, constants.RolePrimary, constants.RolePrimary, 7003, "sdw2", "sdw2", "/data/primary/gpseg1")
	mirror1 = createSegment(t, 4, 0, constants.RoleMirror, constants.RoleMirror, 7004, "sdw2", "sdw2", "/data/mirror/gpseg0")
	mirror2 = createSegment(t, 6, 1, constants.RoleMirror, constants.RoleMirror, 7005, "sdw1", "sdw1", "/data/mirror/gpseg1")

	gparray = greenplum.GpArray{
		Coordinator: coordinator,
		Standby:     standby,
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
}

func TestNewGpArrayFromCatalog(t *testing.T) {
	initializeGpArray(t)

	t.Run("succesfully builds the gparray for mirrorred balanced cluster", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator, standby, primary1, mirror1, primary2, mirror2)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		result, err := greenplum.NewGpArrayFromCatalog(conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := &greenplum.GpArray{
			Coordinator: coordinator,
			Standby:     standby,
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

		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %+v, want %+v", result, expected)
		}
	})

	t.Run("succesfully builds the gparray for mirrorred unbalanced cluster", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		// switch the primary and mirror roles
		primary1.Role = constants.RoleMirror
		primary2.Role = constants.RoleMirror
		mirror1.Role = constants.RolePrimary
		mirror2.Role = constants.RolePrimary
		defer initializeGpArray(t)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator, standby, primary1, mirror1, primary2, mirror2)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		result, err := greenplum.NewGpArrayFromCatalog(conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := &greenplum.GpArray{
			Coordinator: coordinator,
			Standby:     standby,
			SegmentPairs: []greenplum.SegmentPair{
				{
					Primary: mirror1,
					Mirror:  primary1,
				},
				{
					Primary: mirror2,
					Mirror:  primary2,
				},
			},
		}

		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %+v, want %+v", result, expected)
		}
	})

	t.Run("succesfully builds the gparray for mirrorless cluster", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator, primary1, primary2)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		result, err := greenplum.NewGpArrayFromCatalog(conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := &greenplum.GpArray{
			Coordinator: coordinator,
			SegmentPairs: []greenplum.SegmentPair{
				{
					Primary: primary1,
				},
				{
					Primary: primary2,
				},
			},
		}

		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %+v, want %+v", result, expected)
		}
	})

	t.Run("errors out when both primary and mirror have the same role", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		mirror1.Role = constants.RolePrimary
		defer initializeGpArray(t)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator, primary1, mirror1, primary2, mirror2)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		_, err := greenplum.NewGpArrayFromCatalog(conn)

		expectedErrString := fmt.Sprintf("invalid configuration, not a valid segment pair for content %d", mirror1.Content)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("errors out when there are more than 2 segments per content", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		primary2.Content = primary1.Content
		defer initializeGpArray(t)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator, primary1, mirror1, primary2)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		_, err := greenplum.NewGpArrayFromCatalog(conn)

		expectedErrString := "invalid configuration, found more than 2 segments per content"
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("errors out when there is no primary segment found", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator, mirror1)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		_, err := greenplum.NewGpArrayFromCatalog(conn)

		expectedErrString := fmt.Sprintf("invalid configuration, no primary segment found for content %d", mirror1.Content)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("errors out when there is no segment found", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		_, err := greenplum.NewGpArrayFromCatalog(conn)

		expectedErrString := "invalid configuration, no segments found"
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("errors out when there is inconsistent number of segments per content", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator, primary1, mirror1, primary2)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		_, err := greenplum.NewGpArrayFromCatalog(conn)

		expectedErrString := "invalid configuration, number of segments per content is not consistent"
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("errors out when there is invalid segment role", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		coordinator.Role = "invalid"
		defer initializeGpArray(t)

		rows := sqlmock.NewRows([]string{"dbid", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator, primary1, mirror1, primary2)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		_, err := greenplum.NewGpArrayFromCatalog(conn)

		expectedErrString := fmt.Sprintf("invalid configuration for segment with dbid %d", coordinator.Dbid)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("errors out when not able to query the database", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		expectedErr := errors.New("error")
		mock.ExpectQuery("SELECT").WillReturnError(expectedErr)

		_, err := greenplum.NewGpArrayFromCatalog(conn)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		expectedErrPrefix := fmt.Sprintf("failed to get %s", constants.GpSegmentConfiguration)
		if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
			t.Fatalf("got %v, want prefix %q", err, expectedErrPrefix)
		}
	})

	t.Run("errors out when query result does not match segment object", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		rows := sqlmock.NewRows([]string{"undefined_column", "content", "role", "preferredrole", "port", "hostname", "address", "datadir"})
		addSegmentRows(t, rows, coordinator)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		_, err := greenplum.NewGpArrayFromCatalog(conn)

		expectedErrPrefix := fmt.Sprintf("failed to get %s", constants.GpSegmentConfiguration)
		if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
			t.Fatalf("got %v, want prefix %q", err, expectedErrPrefix)
		}
	})
}

func TestRegisterSegments(t *testing.T) {
	t.Run("succesfully registers the coordinator segment", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		seg := &idl.Segment{
			Port:          1234,
			HostName:      "cdw",
			HostAddress:   "cdw",
			DataDirectory: "/data/gpseg-1",
		}
		mock.ExpectExec(regexp.QuoteMeta(fmt.Sprintf("SELECT pg_catalog.gp_add_segment(1::int2, -1::int2, 'p', 'p', 's', 'u', '%d', '%s', '%s', '%s')",
			seg.Port, seg.HostName, seg.HostAddress, seg.DataDirectory))).WillReturnResult(sqlmock.NewResult(1, 1))

		err := greenplum.RegisterCoordinator(seg, conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("succesfully registers the primary segments", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		segs := []*idl.Segment{
			{
				Port:          1111,
				HostName:      "sdw1",
				HostAddress:   "sdw1",
				DataDirectory: "/data/gpseg0",
			},
			{
				Port:          2222,
				HostName:      "sdw2",
				HostAddress:   "sdw2",
				DataDirectory: "/data/gpseg1",
			},
		}

		for _, seg := range segs {
			mock.ExpectExec(regexp.QuoteMeta(fmt.Sprintf("SELECT pg_catalog.gp_add_segment_primary( '%s', '%s', %d, '%s')",
				seg.HostName, seg.HostAddress, seg.Port, seg.DataDirectory))).WillReturnResult(sqlmock.NewResult(1, 1))
		}
		mock.ExpectExec("SET allow_system_table_mods=true; UPDATE gp_segment_configuration SET content = content - 1 where content > 0").WillReturnResult(sqlmock.NewResult(1, 1))

		err := greenplum.RegisterPrimarySegments(segs, conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("succesfully registers the mirror segments", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		segs := []*idl.Segment{
			{
				Contentid:     0,
				Port:          1111,
				HostName:      "sdw1",
				HostAddress:   "sdw1",
				DataDirectory: "/data/gpseg0",
			},
			{
				Contentid:     1,
				Port:          2222,
				HostName:      "sdw2",
				HostAddress:   "sdw2",
				DataDirectory: "/data/gpseg1",
			},
		}

		for _, seg := range segs {
			mock.ExpectExec(regexp.QuoteMeta(fmt.Sprintf("SELECT pg_catalog.gp_add_segment_mirror(%d::int2, '%s', '%s', %d, '%s')",
				seg.Contentid, seg.HostName, seg.HostAddress, seg.Port, seg.DataDirectory))).WillReturnResult(sqlmock.NewResult(1, 1))
		}

		err := greenplum.RegisterMirrorSegments(segs, conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns appropriate error when fails to register the segment", func(t *testing.T) {
		conn, mock := testutils.CreateAndConnectMockDB(t, 1)

		expectedErr := errors.New("error")
		mock.ExpectExec("SELECT").WillReturnError(expectedErr)
		mock.ExpectExec("SELECT").WillReturnError(expectedErr)
		mock.ExpectExec("SELECT").WillReturnError(expectedErr)

		err := greenplum.RegisterCoordinator(&idl.Segment{}, conn)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		err = greenplum.RegisterPrimarySegments([]*idl.Segment{{}}, conn)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		err = greenplum.RegisterMirrorSegments([]*idl.Segment{{}}, conn)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}

func TestGpArray(t *testing.T) {
	initializeGpArray(t)

	t.Run("returns the primary and mirror segments correctly", func(t *testing.T) {
		result := gparray.GetPrimarySegments()
		expected := []greenplum.Segment{*primary1, *primary2}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %+v, want %+v", result, expected)
		}

		result = gparray.GetMirrorSegments()
		expected = []greenplum.Segment{*mirror1, *mirror2}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %+v, want %+v", result, expected)
		}

		result = gparray.GetAllSegments()
		expected = []greenplum.Segment{*primary1, *primary2, *mirror1, *mirror2}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %+v, want %+v", result, expected)
		}
	})

	t.Run("returns the segment pair for the given content", func(t *testing.T) {
		result, err := gparray.GetSegmentPairForContent(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := &greenplum.SegmentPair{
			Primary: primary1,
			Mirror:  mirror1,
		}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %+v, want %+v", result, expected)
		}
	})

	t.Run("returns error when not able to find the pair for a given content", func(t *testing.T) {
		_, err := gparray.GetSegmentPairForContent(-2)

		expectedErrString := "could not find any segments with content -2"
		if err.Error() != expectedErrString {
			t.Fatalf("got %s, want %s", err, expectedErrString)
		}
	})

	t.Run("check if the cluster has mirrors", func(t *testing.T) {
		if !gparray.HasMirrors() {
			t.Fatalf("got no mirrors, want mirrors")
		}

		gparray.SegmentPairs = []greenplum.SegmentPair{
			{
				Primary: primary1,
			},
			{
				Primary: primary2,
			},
		}
		defer initializeGpArray(t)

		if gparray.HasMirrors() {
			t.Fatalf("got mirrors, wnat no mirrors")
		}
	})

	t.Run("get segments by hostname", func(t *testing.T) {
		expected := map[string][]greenplum.Segment{
			primary1.Hostname: {*primary1, *mirror2},
			primary2.Hostname: {*primary2, *mirror1},
		}

		result := gparray.GetSegmentsByHost()
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %+v, want %+v", result, expected)
		}
	})
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

func addSegmentRows(t *testing.T, rows *sqlmock.Rows, segs ...*greenplum.Segment) {
	t.Helper()

	for _, seg := range segs {
		rows.AddRow(seg.Dbid, seg.Content, seg.Role, seg.PreferredRole, seg.Port, seg.Hostname, seg.Address, seg.DataDir)
	}
}
