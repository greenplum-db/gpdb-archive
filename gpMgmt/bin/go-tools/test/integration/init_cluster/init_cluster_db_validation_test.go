package init_cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/cluster"
	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gpdb/gp/cli"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
)

func TestLocaleValidation(t *testing.T) {
	localTypes := []string{"LC_COLLATE", "LC_CTYPE", "LC_MESSAGES", "LC_MONETARY", "LC_NUMERIC", "LC_TIME"}

	t.Run("when LC_ALL is given, it sets the locale for all the types", func(t *testing.T) {
		expected := testutils.GetRandomLocale(t)

		configFile := testutils.GetTempFile(t, "config.json")
		SetConfigKey(t, configFile, "locale", cli.Locale{
			LcAll: expected,
		}, true)

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		for _, localType := range localTypes {
			testutils.AssertPgConfig(t, localType, expected)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("individual locale type takes precedence over LC_ALL", func(t *testing.T) {
		expected := testutils.GetRandomLocale(t)
		expectedLcCtype := testutils.GetRandomLocale(t)

		configFile := testutils.GetTempFile(t, "config.json")
		SetConfigKey(t, configFile, "locale", cli.Locale{
			LcAll:   expected,
			LcCtype: expectedLcCtype,
		}, true)

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		for _, localType := range localTypes {
			if localType == "LC_CTYPE" {
				testutils.AssertPgConfig(t, localType, expectedLcCtype)
			} else {
				testutils.AssertPgConfig(t, localType, expected)
			}
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("when no locale value is provided, inherits the locale from the environment", func(t *testing.T) {
		// TODO: on macos launchd does not inherit the system locale value
		// so skip it for now until we find a way to test it.
		if runtime.GOOS == constants.PlatformDarwin {
			t.Skip()
		}

		configFile := testutils.GetTempFile(t, "config.json")
		UnsetConfigKey(t, configFile, "locale", true)

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		for _, localType := range localTypes {
			testutils.AssertPgConfig(t, localType, testutils.GetSystemLocale(t, localType))
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("when invalid locale is given", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		SetConfigKey(t, configFile, "locale", cli.Locale{
			LcAll: "invalid.locale",
		}, true)

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := `\[ERROR\]:-validating hosts: host: (\S+), locale value 'invalid.locale' is not a valid locale`

		match, err := regexp.MatchString(expectedOut, result.OutputMsg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !match {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})
}

func TestPgConfig(t *testing.T) {
	t.Run("sets the correct config values as provided by the user", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		SetConfigKey(t, configFile, "coordinator-config", map[string]string{
			"max_connections": "15",
		}, true)
		SetConfigKey(t, configFile, "segment-config", map[string]string{
			"max_connections": "10",
		})
		SetConfigKey(t, configFile, "common-config", map[string]string{
			"max_wal_senders": "5",
		})

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		testutils.AssertPgConfig(t, "max_connections", "15", -1)
		testutils.AssertPgConfig(t, "max_connections", "10", 0)
		testutils.AssertPgConfig(t, "max_wal_senders", "5", -1)
		testutils.AssertPgConfig(t, "max_wal_senders", "5", 0)

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("segment-config and coordinator-config take precedence over the common-config", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		SetConfigKey(t, configFile, "coordinator-config", map[string]string{
			"max_connections": "15",
		}, true)
		SetConfigKey(t, configFile, "segment-config", map[string]string{
			"max_connections": "10",
		})
		SetConfigKey(t, configFile, "common-config", map[string]string{
			"max_connections": "25",
		})

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		testutils.AssertPgConfig(t, "max_connections", "15", -1)
		testutils.AssertPgConfig(t, "max_connections", "10", 0)

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("check if the gp_segment_configuration table has the correct value for primary", func(t *testing.T) {
		var value cli.Segment
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)
		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		}

		var primarySegments []cli.Segment
		primarySegments = append(primarySegments, value)

		for _, segPair := range valueSegPair {
			primarySegments = append(primarySegments, *segPair.Primary)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, false)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		resultSegs := make([]cli.Segment, len(segConfigs))
		for i, seg := range segConfigs {
			resultSegs[i] = cli.Segment{
				Hostname:      seg.Hostname,
				Port:          seg.Port,
				DataDirectory: seg.DataDir,
				Address:       seg.Hostname,
			}
		}

		if !reflect.DeepEqual(resultSegs, primarySegments) {
			t.Fatalf("got %+v, want %+v", resultSegs, primarySegments)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("initialize cluster with default config and verify default values used correctly", func(t *testing.T) {
		var expectedOut string
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}
		expectedOutput := result.OutputMsg

		expectedOut = "[INFO]:-Could not find encoding in cluster config, defaulting to UTF-8"
		if !strings.Contains(expectedOutput, expectedOut) {
			t.Fatalf("Output does not contain the expected string.\nExpected: %q\nGot: %q", expectedOut, expectedOutput)
		}

		expectedOut = "[INFO]:-Coordinator max_connections not set, will set to value 150 from CommonConfig"
		if !strings.Contains(expectedOutput, expectedOut) {
			t.Fatalf("Output does not contain the expected string.\nExpected: %q\nGot: %q", expectedOut, expectedOutput)
		}

		expectedOut = "[INFO]:-shared_buffers is not set in CommonConfig, will set to default value 128000kB"
		if !strings.Contains(expectedOutput, expectedOut) {
			t.Fatalf("Output does not contain the expected string.\nExpected: %q\nGot: %q", expectedOut, expectedOutput)
		}

		testutils.AssertPgConfig(t, "max_connections", "150", -1)
		testutils.AssertPgConfig(t, "shared_buffers", "125MB", -1)
		testutils.AssertPgConfig(t, "client_encoding", "UTF8", -1)

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("check if the gp_segment_configuration table has the correct value for mirrors", func(t *testing.T) {
		var value cli.Segment
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}

		mirrorSegs := config.Get("segment-array")
		valueSegPair, ok := mirrorSegs.([]cli.SegmentPair)
		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", mirrorSegs)
		}

		var mirrorSegments []cli.Segment

		for _, segPair := range valueSegPair {
			mirrorSegments = append(mirrorSegments, *segPair.Mirror)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, false, true)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		resultSegs := make([]cli.Segment, len(segConfigs))
		for i, seg := range segConfigs {
			resultSegs[i] = cli.Segment{
				Hostname:      seg.Hostname,
				Port:          seg.Port,
				DataDirectory: seg.DataDir,
				Address:       seg.Hostname,
			}
		}

		if !reflect.DeepEqual(resultSegs, mirrorSegments) {
			t.Fatalf("got %+v, want %+v", resultSegs, mirrorSegments)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCollations(t *testing.T) {
	t.Run("collations are imported successfully", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := "[INFO]:-Importing system collations"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		// before importing collations
		testutils.ExecQuery(t, "", "CREATE TABLE collationimport1 AS SELECT * FROM pg_collation WHERE collnamespace = 'pg_catalog'::regnamespace")

		// importing collations
		rows := testutils.ExecQuery(t, "", "SELECT pg_import_system_collations('pg_catalog')")
		testutils.AssertRowCount(t, rows, 1)

		// after importing collations
		testutils.ExecQuery(t, "", "CREATE TABLE collationimport2 AS SELECT * FROM pg_collation WHERE collnamespace = 'pg_catalog'::regnamespace")

		// there should be no difference before and after
		rows = testutils.ExecQuery(t, "", "SELECT * FROM collationimport1 EXCEPT SELECT * FROM collationimport2")
		testutils.AssertRowCount(t, rows, 0)

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDbCreationValidation(t *testing.T) {
	testDatabaseCreation := func(t *testing.T, dbName string) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		SetConfigKey(t, configFile, "db-name", dbName, true)

		InitClusterResult, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", InitClusterResult.OutputMsg, err)
		}

		rows := testutils.ExecQuery(t, "", "SELECT datname FROM pg_database")
		defer rows.Close()
		foundDB := false
		for rows.Next() {
			var db string
			if err := rows.Scan(&db); err != nil {
				t.Fatalf("unexpected error scanning result: %v", err)
			}
			if db == dbName {
				foundDB = true
				break
			}
		}
		if !foundDB {
			t.Fatalf("Database %s should exist after creating it", dbName)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	t.Run("validate cluster creation by specifying db name", func(t *testing.T) {
		testDatabaseCreation(t, "testdb")
	})

	t.Run("validate cluster creation by specifying hyphen in db name", func(t *testing.T) {
		testDatabaseCreation(t, "test-db")
	})

	t.Run("validate default databases creation when no db name is specified", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		foundDBs := make(map[string]bool)

		rows := testutils.ExecQuery(t, "", "SELECT datname from pg_database")
		for rows.Next() {
			var dbName string
			if err := rows.Scan(&dbName); err != nil {
				t.Fatalf("unexpected error scanning result: %v", err)
			}
			foundDBs[dbName] = true
		}

		expectedDBs := []string{"postgres", "template1", "template0"}
		for _, db := range expectedDBs {
			if !foundDBs[db] {
				t.Fatalf("Default database %s should exist after creating it", db)
			}
		}

		if len(foundDBs) != len(expectedDBs) {
			t.Fatalf("Unexpected number of databases found: expected %d, found %d", len(expectedDBs), len(foundDBs))
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

}

func TestGpToolKitValidation(t *testing.T) {
	t.Run("check if the gp_toolkit extension is created", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		QueryResult := testutils.ExecQuery(t, "", "select extname from pg_extension ")
		foundGpToolkit := false
		for QueryResult.Next() {
			var extName string
			err := QueryResult.Scan(&extName)
			if err != nil {
				t.Fatalf("unexpected error scanning result: %v", err)
			}
			if extName == "gp_toolkit" {
				foundGpToolkit = true
				break
			}
		}

		// Validate that "gp_toolkit" is present
		if !foundGpToolkit {
			t.Fatalf("Extension 'gp_toolkit' should exist in pg_extension")
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
func TestPgStatReplicationValidation(t *testing.T) {
	t.Run("check if the pg_stat_replication table has the correct number of primary hosts", func(t *testing.T) {
		var value cli.Segment
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)
		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		}
		numPrimary := len(valueSegPair)

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		rows := testutils.ExecQuery(t, "", "select * from gp_stat_replication where state='streaming'")
		testutils.AssertRowCount(t, rows, numPrimary)

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGpRecoverSegValidation(t *testing.T) {
	t.Run("check if the cluster is created successfully and verify that gprecoverseg works fine", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)
		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		MirrorHostName := valueSegPair[0].Mirror.Hostname
		MirroDataDirectory := valueSegPair[0].Mirror.DataDirectory

		err = testutils.RunGpStopSegment(MirroDataDirectory, MirrorHostName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testutils.WaitForDesiredQueryResult(t, "", fmt.Sprintf("select status from gp_segment_configuration where role ='m' and datadir='%s'", valueSegPair[0].Mirror.DataDirectory), "d")

		result, err = testutils.RunGpRecoverSeg()
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut = "[INFO]:-Segments successfully recovered"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestExpansionValidation(t *testing.T) {
	t.Run("check if primary ports are adjusted as per coordinator port in all the hosts when not specified", func(t *testing.T) {
		var value cli.Segment
		var ok bool
		var primarySegConfigs []cluster.SegConfig

		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}
		coordinatorPort := value.Port

		configMap := config.AllSettings()
		delete(configMap, "primary-base-port")
		encodedConfig, err := json.MarshalIndent(configMap, "", " ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = os.WriteFile(configFile, encodedConfig, 0777)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedWarning := fmt.Sprintf("[WARNING]:-primary-base-port value not specified. Setting default to: %d", coordinatorPort+2)
		if !strings.Contains(result.OutputMsg, expectedWarning) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedWarning)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, false)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		hostList := (config.GetStringSlice("hostlist"))

		for _, hostname := range hostList {
			for _, seg := range segConfigs {
				if seg.ContentID != -1 && seg.Role == "p" && seg.Hostname == hostname {
					primarySegConfigs = append(primarySegConfigs, seg)
				}
			}
			for i, seg := range primarySegConfigs {
				expectedPrimaryPort := coordinatorPort + 2 + i
				if seg.Port != expectedPrimaryPort {
					t.Fatalf("Primary port mismatch for segment %s. Got: %d, Expected: %d", seg.Hostname, seg.Port, expectedPrimaryPort)
				}
			}
			primarySegConfigs = nil
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("check if mirror base ports are adjusted as per coordinator port in all the hosts when not specified", func(t *testing.T) {
		var value cli.Segment
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}
		coordinatorPort := value.Port

		configMap := config.AllSettings()
		delete(configMap, "mirror-base-port")
		encodedConfig, err := json.MarshalIndent(configMap, "", " ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = os.WriteFile(configFile, encodedConfig, 0777)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedWarning := fmt.Sprintf("[WARNING]:-mirror-base-port value not specified. Setting default to: %d", coordinatorPort+1002)
		if !strings.Contains(result.OutputMsg, expectedWarning) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedWarning)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, false, true)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		var mirrorSegConfigs []cluster.SegConfig
		hostList := (config.GetStringSlice("hostlist"))

		for _, hostname := range hostList {
			for _, seg := range segConfigs {
				if seg.ContentID != -1 && seg.Role == "m" && seg.Hostname == hostname {
					mirrorSegConfigs = append(mirrorSegConfigs, seg)
				}
			}
			for i, seg := range mirrorSegConfigs {
				expectedMirrorPort := coordinatorPort + 1002 + i
				if seg.Port != expectedMirrorPort {
					t.Fatalf("Mirror port mismatch for segment %s. Got: %d, Expected: %d", seg.Hostname, seg.Port, expectedMirrorPort)
				}
			}
			mirrorSegConfigs = nil
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("verify expansion with mirror port specified", func(t *testing.T) {
		var value cli.Segment
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}
		coordinatorPort := value.Port

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, false, true)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		var mirrorSegConfigs []cluster.SegConfig
		hostList := (config.GetStringSlice("hostlist"))

		for _, hostname := range hostList {
			for _, seg := range segConfigs {
				if seg.ContentID != -1 && seg.Role == "m" && seg.Hostname == hostname {
					mirrorSegConfigs = append(mirrorSegConfigs, seg)
				}
			}
			for i, seg := range mirrorSegConfigs {
				expectedMirrorPort := coordinatorPort + 1002 + i
				if seg.Port != expectedMirrorPort {
					t.Fatalf("Mirror port mismatch for segment %s. Got: %d, Expected: %d", seg.Hostname, seg.Port, expectedMirrorPort)
				}
			}
			mirrorSegConfigs = nil
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("verify expansion by initialize cluster with default config and verify default values used correctly", func(t *testing.T) {
		var expectedOut string
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}
		expectedOutput := result.OutputMsg

		expectedOut = "[INFO]:-Could not find encoding in cluster config, defaulting to UTF-8"
		if !strings.Contains(expectedOutput, expectedOut) {
			t.Fatalf("Output does not contain the expected string.\nExpected: %q\nGot: %q", expectedOut, expectedOutput)
		}

		expectedOut = "[INFO]:-Coordinator max_connections not set, will set to value 150 from CommonConfig"
		if !strings.Contains(expectedOutput, expectedOut) {
			t.Fatalf("Output does not contain the expected string.\nExpected: %q\nGot: %q", expectedOut, expectedOutput)
		}

		expectedOut = "[INFO]:-shared_buffers is not set in CommonConfig, will set to default value 128000kB"
		if !strings.Contains(expectedOutput, expectedOut) {
			t.Fatalf("Output does not contain the expected string.\nExpected: %q\nGot: %q", expectedOut, expectedOutput)
		}

		testutils.AssertPgConfig(t, "max_connections", "150", -1)
		testutils.AssertPgConfig(t, "shared_buffers", "125MB", -1)
		testutils.AssertPgConfig(t, "client_encoding", "UTF8", -1)

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validate expansion that proper number of primary and mirror directories are created in each hosts", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
		primaryDirs := len(config.GetStringSlice("primary-data-directories"))
		mirrorDirs := len(config.GetStringSlice("mirror-data-directories"))
		hostList := len(config.GetStringSlice("hostlist"))

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, true)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		var primaryDataDirs []string
		var mirrorDataDirs []string

		for _, seg := range segConfigs {
			if seg.ContentID == -1 {
				continue
			} else if seg.Role == "p" {
				primaryDataDirs = append(primaryDataDirs, seg.DataDir)
			} else if seg.Role == "m" {
				mirrorDataDirs = append(mirrorDataDirs, seg.DataDir)
			}
		}

		actualPrimaryCount := len(primaryDataDirs)
		actualMirrorCount := len(mirrorDataDirs)

		expectedPrimaryCount := primaryDirs * hostList
		expectedMirrorCount := mirrorDirs * hostList

		if actualPrimaryCount != expectedPrimaryCount {
			t.Fatalf("Error: Primary data directories count mismatch: expected %d, got %d", expectedPrimaryCount, actualPrimaryCount)
		}

		if actualMirrorCount != expectedMirrorCount {
			t.Fatalf("Error: Mirror data directories count mismatch: expected %d, got %d", expectedMirrorCount, actualMirrorCount)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validate the group mirroring and check if segments are distributed properly across hosts", func(t *testing.T) {
		hostList := testutils.GetHostListFromFile(*hostfile)

		if len(hostList) == 1 {
			t.Skip()
		}
		var value cli.Segment
		var ok bool

		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, true)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		hostname := config.Get("hostlist").([]string)[0]
		primaries := make(map[int][]cluster.SegConfig)
		for _, seg := range segConfigs {
			if seg.ContentID != -1 && seg.Role == "p" && seg.Hostname == hostname {
				primaries[seg.ContentID] = append(primaries[seg.ContentID], seg)
			}
		}
		mirrors := make(map[int][]cluster.SegConfig)
		for _, seg := range segConfigs {
			if seg.Role == "m" {
				if primary, ok := primaries[seg.ContentID]; ok {
					mirrors[primary[0].ContentID] = append(mirrors[primary[0].ContentID], seg)
				}
			}
		}

		var mirrorHostnames []string
		seen := make(map[string]bool)
		var primaryHostnames []string

		for _, configs := range mirrors {
			for _, config := range configs {
				mirrorHostnames = append(mirrorHostnames, config.Hostname)
				seen[config.Hostname] = true
			}
		}
		for _, configs := range primaries {
			for _, config := range configs {
				primaryHostnames = append(primaryHostnames, config.Hostname)
			}
		}
		for _, mirrorHostname := range mirrorHostnames {
			for _, primaryHostname := range primaryHostnames {
				if mirrorHostname == primaryHostname {
					t.Fatalf("Error: Mirrors are hosted on the same host as primary: %s", mirrorHostname)
				}
			}
		}
		if len(seen) > 1 {
			t.Fatalf("Error: Group mirroing validation Failed: All hostnames are not same for mirrors: %v", mirrorHostnames)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("validate the spread mirroring and check if segments are distributed properly across hosts", func(t *testing.T) {
		hostList := testutils.GetHostListFromFile(*hostfile)
		if len(hostList) == 1 {
			t.Skip()
		}
		var value cli.Segment
		var ok bool

		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}

		config.Set("mirroring-type", "spread")
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, true)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		hostname := config.Get("hostlist").([]string)[0]
		primaries := make(map[int][]cluster.SegConfig)
		for _, seg := range segConfigs {
			if seg.ContentID != -1 && seg.Role == "p" && seg.Hostname == hostname {
				primaries[seg.ContentID] = append(primaries[seg.ContentID], seg)
			}
		}

		mirrors := make(map[int][]cluster.SegConfig)
		for _, seg := range segConfigs {
			if seg.Role == "m" {
				if primary, ok := primaries[seg.ContentID]; ok {
					mirrors[primary[0].ContentID] = append(mirrors[primary[0].ContentID], seg)
				}
			}
		}

		var mirrorHostnames []string
		seen := make(map[string]bool)
		var primaryHostnames []string

		for _, configs := range mirrors {
			for _, config := range configs {
				mirrorHostnames = append(mirrorHostnames, config.Hostname)
				seen[config.Hostname] = true
			}
		}
		for _, configs := range primaries {
			for _, config := range configs {
				primaryHostnames = append(primaryHostnames, config.Hostname)
			}
		}
		for _, mirrorHostname := range mirrorHostnames {
			for _, primaryHostname := range primaryHostnames {
				if mirrorHostname == primaryHostname {
					t.Fatalf("Error: Mirrors are hosted on the same host as primary: %s", mirrorHostname)
				}
			}
		}
		if len(seen) != len(mirrorHostnames) {
			t.Fatalf("Error: Spread mirroing Validation Failed, Hostnames are not different. %v", mirrorHostnames)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})
	t.Run("validate that the expansion creates data directories on different locations as specified in primary and mirror base-directories", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		conn := dbconn.NewDBConnFromEnvironment("postgres")
		if err := conn.Connect(1); err != nil {
			t.Fatalf("Error connecting to the database: %v", err)
		}
		defer conn.Close()

		segConfigs, err := cluster.GetSegmentConfiguration(conn, true)
		if err != nil {
			t.Fatalf("Error getting segment configuration: %v", err)
		}

		primaryBaseDirs := config.GetStringSlice("primary-data-directories")
		mirrorBaseDirs := config.GetStringSlice("mirror-data-directories")
		for _, seg := range segConfigs {
			if seg.ContentID == -1 {
				continue
			}
			dir := filepath.Dir(seg.DataDir)
			var baseDirs []string
			if seg.Role == "p" {
				baseDirs = primaryBaseDirs
			} else if seg.Role == "m" {
				baseDirs = mirrorBaseDirs
			} else {
				continue
			}
			matched := false
			for _, baseDir := range baseDirs {
				if dir == baseDir {
					matched = true
					break
				}
			}
			if !matched {
				t.Fatalf("Segment directory %s is not created under any of the specified %s directories", seg.DataDir, seg.Role)
			}
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

}
