package postgres_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/utils"
	"github.com/greenplum-db/gpdb/gp/utils/postgres"
)

func TestUpdatePostgresqlConf(t *testing.T) {
	testhelper.SetupTestLogger()

	cases := []struct {
		overwrite    bool
		configParams map[string]string
		confContent  string
		expected     string
	}{
		{
			overwrite: true,
			configParams: map[string]string{
				"guc_1": "new_value_1",
			},
			confContent: `
guc_1 = value_1
guc_2 = value_2
# comment`,
			expected: `
guc_1 = 'new_value_1'
guc_2 = value_2
# comment`,
		},
		{ // when there is duplicated entries, updates all of them
			overwrite: true,
			configParams: map[string]string{
				"guc_1": "new_value_1",
			},
			confContent: `
guc_1 = value_1
guc_2 = value_2
guc_1 = value_1
# comment`,
			expected: `
guc_1 = 'new_value_1'
guc_2 = value_2
guc_1 = 'new_value_1'
# comment`,
		},
		{
			overwrite: false,
			configParams: map[string]string{
				"guc_1": "new_value_1",
			},
			confContent: `
guc_1 = value_1
guc_2 = value_2
# comment`,
			expected: `
guc_1 = 'new_value_1' # guc_1 = value_1
guc_2 = value_2
# comment`,
		},
		{
			overwrite: true,
			configParams: map[string]string{
				"guc_1": "new_value_1",
				"guc_3": "new_value_3",
			},
			confContent: `
guc_1 = value_1
guc_2 = value_2`,
			expected: `
guc_1 = 'new_value_1'
guc_2 = value_2
guc_3 = 'new_value_3'`,
		},
		{
			overwrite: true,
			configParams: map[string]string{
				"guc_1": "new_value_1",
				"guc_3": "1234",
			},
			confContent: `
guc_1 value_1
guc_2 value_2`,
			expected: `
guc_1 = 'new_value_1'
guc_2 value_2
guc_3 = 1234`,
		},
		{
			overwrite: true,
			configParams: map[string]string{
				"guc_1": "new_value_1",
			},
			confContent: `
guc_1  =    value_1
guc_1       value_1
#guc_1 value_1
aguc_1 value_1
a_guc_1 value_1
guc_1a value_1
guc_1_a value_1
guc_1a = value_1
guc_1_a=value_1
guc_2 value_2`,
			expected: `
guc_1 = 'new_value_1'
guc_1 = 'new_value_1'
#guc_1 value_1
aguc_1 value_1
a_guc_1 value_1
guc_1a value_1
guc_1_a value_1
guc_1a = value_1
guc_1_a=value_1
guc_2 value_2`,
		},
	}

	for _, tc := range cases {
		t.Run("correctly updates the postgresql.conf file", func(t *testing.T) {
			dname, confPath := createTempConfFile(t, "postgresql.conf", tc.confContent, 0644)
			defer os.RemoveAll(dname)

			err := postgres.UpdatePostgresqlConf(dname, tc.configParams, tc.overwrite)
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}

			testutils.AssertFileContents(t, confPath, tc.expected)
		})
	}

	t.Run("errors out when there is no file present", func(t *testing.T) {
		dname, _ := createTempConfFile(t, "", "", 0644)
		defer os.RemoveAll(dname)

		expectedErr := os.ErrNotExist
		err := postgres.UpdatePostgresqlConf(dname, map[string]string{}, true)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})

	t.Run("returns appropriate error when fails to update the conf file", func(t *testing.T) {
		dname, _ := createTempConfFile(t, "postgresql.conf", "", 0644)
		defer os.RemoveAll(dname)

		expectedErr := errors.New("error")
		utils.System.Create = func(name string) (*os.File, error) {
			return nil, expectedErr
		}
		defer utils.ResetSystemFunctions()

		err := postgres.UpdatePostgresqlConf(dname, map[string]string{}, true)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}

func TestUpdatePostgresInternalConf(t *testing.T) {
	testhelper.SetupTestLogger()

	t.Run("successfully creates the internal.auto.conf", func(t *testing.T) {
		dname, _ := createTempConfFile(t, "", "", 0644)
		defer os.RemoveAll(dname)

		confPath := filepath.Join(dname, "internal.auto.conf")
		_, err := os.Stat(confPath)
		if !os.IsNotExist(err) {
			t.Fatalf("expected %s to not exist", confPath)
		}

		err = postgres.UpdatePostgresInternalConf(dname, -1)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		expected := "gp_dbid = -1"
		testutils.AssertFileContents(t, confPath, expected)
	})

	t.Run("errors out when not able to create the file", func(t *testing.T) {
		dname, _ := createTempConfFile(t, "internal.auto.conf", "", 0000)
		defer os.RemoveAll(dname)

		expectedErr := os.ErrPermission
		err := postgres.UpdatePostgresInternalConf(dname, -1)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})

	t.Run("errors out when not able to write to the file", func(t *testing.T) {
		dname, _ := createTempConfFile(t, "", "", 0644)
		defer os.RemoveAll(dname)
		utils.System.OpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
			_, writer, _ := os.Pipe()
			writer.Close()

			return writer, nil
		}
		defer utils.ResetSystemFunctions()

		expectedErr := os.ErrClosed
		err := postgres.UpdatePostgresInternalConf(dname, -1)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}

func TestGetConfigValue(t *testing.T) {
	t.Run("correctly returns the config value", func(t *testing.T) {
		utils.System.Open = func(name string) (*os.File, error) {
			reader, writer, _ := os.Pipe()
			defer writer.Close()

			content := `# comment 1
# guc1 = some_value1
invalid_config
guc1 = value1 # guc1 = some_value2
guc2 = value2
guc1 = latest_value
guc3          value3
guc4         =   value4
`
			_, err := writer.WriteString(content)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			return reader, nil
		}
		defer utils.ResetSystemFunctions()

		cases := map[string]string{
			"guc1": "latest_value",
			"guc3": "value3",
			"guc4": "value4",
		}

		for config, expected := range cases {
			result, err := postgres.GetConfigValue("gpseg", config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != expected {
				t.Fatalf("got %s, want %s", result, expected)
			}
		}
	})

	t.Run("returns error when not able to find the config value", func(t *testing.T) {
		utils.System.Open = func(name string) (*os.File, error) {
			reader, writer, _ := os.Pipe()
			defer writer.Close()

			return reader, nil
		}
		defer utils.ResetSystemFunctions()

		_, err := postgres.GetConfigValue("gpseg", "config")
		expectedErrString := `did not find any config parameter named "config" in gpseg/postgresql.conf`
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("returns error when not able to open the file", func(t *testing.T) {
		expectedErr := errors.New("error")
		utils.System.Open = func(name string) (*os.File, error) {
			return nil, expectedErr
		}
		defer utils.ResetSystemFunctions()

		_, err := postgres.GetConfigValue("gpseg", "config")
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}

func createTempConfFile(t *testing.T, filename, content string, perm fs.FileMode) (string, string) {
	t.Helper()

	dname, err := os.MkdirTemp("", "gpseg")
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	filepath := filepath.Join(dname, filename)
	if filename != "" {
		err := os.WriteFile(filepath, []byte(content), perm)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
	}

	return dname, filepath
}
