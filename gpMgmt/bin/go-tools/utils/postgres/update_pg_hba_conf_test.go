package postgres_test

import (
	"errors"
	"os"
	"os/user"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/postgres"
)

func TestBuildPgHbaConf(t *testing.T) {
	testhelper.SetupTestLogger()

	cases := []struct {
		coordinator      bool
		coordinatorAddrs []string
		addrs            []string
		confContent      string
		expected         string
	}{
		{
			coordinator: true,
			addrs:       []string{"192.0.1.0/24", "2001:db8::/32"},
			confContent: ``,
			expected: `local	all	gpadmin	ident
local	replication	gpadmin	ident
host	all	gpadmin	localhost	trust
host	all	gpadmin	192.0.1.0/24	trust
host	all	gpadmin	2001:db8::/32	trust
host	replication	gpadmin	samehost	trust
host	replication	gpadmin	localhost	trust
host	replication	gpadmin	192.0.1.0/24	trust
host	replication	gpadmin	2001:db8::/32	trust`,
		},
		{
			coordinator: true,
			addrs:       []string{"cdw"},
			confContent: ``,
			expected: `local	all	gpadmin	ident
local	replication	gpadmin	ident
host	all	gpadmin	localhost	trust
host	all	gpadmin	cdw	trust
host	replication	gpadmin	samehost	trust
host	replication	gpadmin	localhost	trust
host	replication	gpadmin	cdw	trust`,
		},
		{
			coordinator: true,
			addrs:       []string{"cdw"},
			confContent: `# foo
foobar
# bar`,
			expected: `# foo
# bar
local	all	gpadmin	ident
local	replication	gpadmin	ident
host	all	gpadmin	localhost	trust
host	all	gpadmin	cdw	trust
host	replication	gpadmin	samehost	trust
host	replication	gpadmin	localhost	trust
host	replication	gpadmin	cdw	trust`,
		},
		{
			coordinator:      false,
			coordinatorAddrs: []string{"192.0.0.0/24"},
			addrs:            []string{"192.0.1.0/24", "2001:db8::/32"},
			confContent:      ``,
			expected: `host	all	all	192.0.0.0/24	trust
host	all	gpadmin	192.0.1.0/24	trust
host	all	gpadmin	2001:db8::/32	trust`,
		},
		{
			coordinator:      false,
			coordinatorAddrs: []string{"cdw"},
			addrs:            []string{"sdw"},
			confContent:      ``,
			expected: `host	all	all	cdw	trust
host	all	gpadmin	sdw	trust`,
		},
		{
			coordinator:      false,
			coordinatorAddrs: []string{"cdw"},
			addrs:            []string{"sdw"},
			confContent: `# foo
foobar
# bar`,
			expected: `# foo
foobar
# bar
host	all	all	cdw	trust
host	all	gpadmin	sdw	trust`,
		},
		{ // removes duplicate entries ignoring the comments
			coordinator:      false,
			coordinatorAddrs: []string{"cdw"},
			addrs:            []string{"sdw"},
			confContent: `# foo
# foo
      host all all cdw trust
host	all	gpadmin	sdw	trust
host	all	gpadmin	sdw	trust`,
			expected: `# foo
# foo
host	all	all	cdw	trust
host	all	gpadmin	sdw	trust`,
		},
	}

	for _, tc := range cases {
		t.Run("correctly builds the pg_hba.conf file", func(t *testing.T) {
			dname, confPath := createTempConfFile(t, "pg_hba.conf", tc.confContent, 0644)
			defer os.RemoveAll(dname)

			utils.System.CurrentUser = func() (*user.User, error) {
				return &user.User{Username: "gpadmin"}, nil
			}
			defer utils.ResetSystemFunctions()

			var err error
			if tc.coordinator {
				err = postgres.BuildCoordinatorPgHbaConf(dname, tc.addrs)
			} else {
				err = postgres.UpdateSegmentPgHbaConf(dname, tc.addrs, false, tc.coordinatorAddrs...)
			}

			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}

			testutils.AssertFileContents(t, confPath, tc.expected)
		})
	}

	failureCases := []struct {
		coordinator bool
	}{
		{
			coordinator: true,
		},
		{
			coordinator: false,
		},
	}

	for _, tc := range failureCases {
		t.Run("errors out when there is no file present", func(t *testing.T) {
			dname, _ := createTempConfFile(t, "", "", 0644)
			defer os.RemoveAll(dname)

			var err error
			expectedErr := os.ErrNotExist
			if tc.coordinator {
				err = postgres.BuildCoordinatorPgHbaConf(dname, []string{"cdw"})
			} else {
				err = postgres.UpdateSegmentPgHbaConf(dname, []string{"sdw"}, false)
			}

			if !errors.Is(err, expectedErr) {
				t.Fatalf("got %#v, want %#v", err, expectedErr)
			}
		})

		t.Run("errors out when not able to get the current user", func(t *testing.T) {
			dname, _ := createTempConfFile(t, "pg_hba.conf", "", 0644)
			defer os.RemoveAll(dname)

			expectedErr := errors.New("error")
			utils.System.CurrentUser = func() (*user.User, error) {
				return nil, expectedErr
			}
			defer utils.ResetSystemFunctions()

			var err error
			if tc.coordinator {
				err = postgres.BuildCoordinatorPgHbaConf(dname, []string{"cdw"})
			} else {
				err = postgres.UpdateSegmentPgHbaConf(dname, []string{"sdw"}, false)
			}

			if !errors.Is(err, expectedErr) {
				t.Fatalf("got %#v, want %#v", err, expectedErr)
			}
		})

		t.Run("errors out when fails to update the conf file", func(t *testing.T) {
			dname, _ := createTempConfFile(t, "pg_hba.conf", "", 0644)
			defer os.RemoveAll(dname)

			expectedErr := errors.New("error")
			utils.System.Create = func(name string) (*os.File, error) {
				return nil, expectedErr
			}
			defer utils.ResetSystemFunctions()

			var err error
			if tc.coordinator {
				err = postgres.BuildCoordinatorPgHbaConf(dname, []string{"cdw"})
			} else {
				err = postgres.UpdateSegmentPgHbaConf(dname, []string{"sdw"}, false)
			}

			if !errors.Is(err, expectedErr) {
				t.Fatalf("got %#v, want %#v", err, expectedErr)
			}
		})
	}
}
