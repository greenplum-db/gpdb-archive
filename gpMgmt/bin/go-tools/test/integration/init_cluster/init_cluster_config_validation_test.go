package init_cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/greenplum-db/gpdb/gp/cli"
	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
	"github.com/spf13/viper"
)

func TestInputFileValidation(t *testing.T) {
	t.Run("cluster creation fails when provided input file doesn't exist", func(t *testing.T) {
		result, err := testutils.RunInitCluster("non_existing_file.json")
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-stat non_existing_file.json: no such file or directory"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when the config file is not provided as an input", func(t *testing.T) {
		result, err := testutils.RunInitCluster()
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-please provide config file for cluster initialization"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when invalid number of arguments are given", func(t *testing.T) {
		result, err := testutils.RunInitCluster("abc", "xyz")
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-more arguments than expected"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("cluster creation fails when provided input file has invalid keys", func(t *testing.T) {
		content := `{
			"invalid_key": "value"
		}
		`
		configFile := testutils.GetTempFile(t, "config.json")
		err := os.WriteFile(configFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := `(?s)\[ERROR\]:-while unmarshaling config file: (.*?) has invalid keys: invalid_key`
		match, err := regexp.MatchString(expectedOut, result.OutputMsg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !match {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("cluster creation fails when provided input file has invalid syntax", func(t *testing.T) {
		content := `{
			$$"key": "value"###
		}
		`
		configFile := testutils.GetTempFile(t, "config.json")
		err := os.WriteFile(configFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-while reading config file:"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("cluster creation fails when input file doesn't have coordinator details", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		UnsetConfigKey(t, configFile, "coordinator", true)

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-no coordinator segment provided in input config file"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("cluster creation fails when the host does not have gp services configured", func(t *testing.T) {
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

		SetConfigKey(t, configFile, "coordinator", cli.Segment{
			Hostname:      "invalid",
			Address:       value.Address,
			Port:          value.Port,
			DataDirectory: value.DataDirectory,
		}, true)

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-following hostnames [invalid] do not have gp services configured. Please configure the services"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("cluster creation fails when input file does not have segment array details", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		UnsetConfigKey(t, configFile, "segment-array", true)

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-no primary segments are provided in input config file"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when encoding is unsupported", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		SetConfigKey(t, configFile, "encoding", "SQL_ASCII", true)

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-SQL_ASCII is no longer supported as a server encoding"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when same data directory is given for a host", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[1].Primary.Hostname = valueSegPair[0].Primary.Hostname
			valueSegPair[1].Primary.Address = valueSegPair[0].Primary.Address
			valueSegPair[0].Primary.DataDirectory = "gpseg1"
			valueSegPair[1].Primary.DataDirectory = "gpseg1"
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-duplicate data directory entry gpseg1 found for host %s", valueSegPair[0].Primary.Hostname)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when same port is given for a host address", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[1].Primary.Hostname = valueSegPair[0].Primary.Hostname
			valueSegPair[1].Primary.Address = valueSegPair[0].Primary.Address
			valueSegPair[0].Primary.Port = 1234
			valueSegPair[1].Primary.Port = 1234
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-duplicate port entry 1234 found for host %s", valueSegPair[1].Primary.Hostname)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when empty data directory is given for a host", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Primary.DataDirectory = ""
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-data_directory has not been provided for segment with hostname %s and port %d", valueSegPair[0].Primary.Hostname, valueSegPair[0].Primary.Port)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when host address is not provided", func(t *testing.T) {
		var ok bool
		var value cli.Segment
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", coordinator)
		}

		value.Address = ""
		SetConfigKey(t, configFile, "coordinator", value, true)

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := fmt.Sprintf("[WARNING]:-hostAddress has not been provided, populating it with same as hostName %s for the segment with port %d and data_directory %s", value.Hostname, value.Port, value.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("when both hostaddress and hostnames are not provided for primary segment", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Primary.Hostname = ""
			valueSegPair[0].Primary.Address = ""
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-hostName has not been provided for the segment with port %d and data_directory %s", valueSegPair[0].Primary.Port, valueSegPair[0].Primary.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when the hostname alone is empty for primary segment", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Primary.Hostname = ""
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-hostName has not been provided for the segment with port %d and data_directory %s", valueSegPair[0].Primary.Port, valueSegPair[0].Primary.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when port number is not provided", func(t *testing.T) {
		var ok bool
		var value cli.Segment
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", coordinator)
		}

		value.Port = 0
		SetConfigKey(t, configFile, "coordinator", value, true)

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-invalid port has been provided for segment with hostname %s and data_directory %s", value.Hostname, value.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when port number is not provided for the primary segment", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t, true)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Primary.Port = 0
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-invalid port has been provided for segment with hostname %s and data_directory %s", valueSegPair[0].Primary.Hostname, valueSegPair[0].Primary.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when number of primary and mirror segments are not equal", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		}

		valueSegPair[0].Mirror = nil
		SetConfigKey(t, configFile, "segment-array", valueSegPair, true)

		numPrimary := len(valueSegPair)
		numMirror := 0
		for _, pair := range valueSegPair {
			if pair.Mirror != nil {
				numMirror++
			}
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-number of primary segments %d and number of mirror segments %d must be equal\n", numPrimary, numMirror)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when the hostname is empty for mirror segment", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Mirror.Hostname = ""
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-hostName has not been provided for the segment with port %d and data_directory %s", valueSegPair[0].Mirror.Port, valueSegPair[0].Mirror.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when port number is not provided for the mirror segment", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Mirror.Port = 0
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-invalid port has been provided for segment with hostname %s and data_directory %s", valueSegPair[0].Mirror.Hostname, valueSegPair[0].Mirror.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when both hostaddress and hostnames are not provided for mirror segment", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Mirror.Hostname = ""
			valueSegPair[0].Mirror.Address = ""
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-hostName has not been provided for the segment with port %d and data_directory %s", valueSegPair[0].Mirror.Port, valueSegPair[0].Mirror.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when empty data directory is given for a mirror segment", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Mirror.DataDirectory = ""
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-data_directory has not been provided for segment with hostname %s and port %d", valueSegPair[0].Mirror.Hostname, valueSegPair[0].Mirror.Port)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when same port is given for a mirror segments", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[1].Mirror.Hostname = valueSegPair[0].Mirror.Hostname
			valueSegPair[1].Mirror.Address = valueSegPair[0].Mirror.Address
			valueSegPair[0].Mirror.Port = 1234
			valueSegPair[1].Mirror.Port = 1234
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-duplicate port entry 1234 found for host %s", valueSegPair[1].Mirror.Hostname)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when same data directory is given for a mirror segment", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[1].Mirror.Hostname = valueSegPair[0].Mirror.Hostname
			valueSegPair[1].Mirror.Address = valueSegPair[0].Mirror.Address
			valueSegPair[0].Mirror.DataDirectory = "gpseg1"
			valueSegPair[1].Mirror.DataDirectory = "gpseg1"
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-duplicate data directory entry gpseg1 found for host %s", valueSegPair[0].Mirror.Hostname)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when hostaddress is empty for the primary and mirror segments", func(t *testing.T) {
		var ok bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		originalPrimaryAddress := valueSegPair[0].Primary.Address

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		} else {
			valueSegPair[0].Primary.Address = ""
			SetConfigKey(t, configFile, "segment-array", valueSegPair, true)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		expectedOut := fmt.Sprintf("[WARNING]:-hostAddress has not been provided, populating it with same as hostName %s for the segment with port %d and data_directory %s", valueSegPair[0].Primary.Hostname, valueSegPair[0].Primary.Port, valueSegPair[0].Primary.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		valueSegPair[0].Primary.Address = originalPrimaryAddress
		SetConfigKey(t, configFile, "segment-array", valueSegPair, true)

		// validation for mirror segments
		valueSegPair[0].Mirror.Address = ""
		SetConfigKey(t, configFile, "segment-array", valueSegPair, true)

		resultMirror, errMirror := testutils.RunInitCluster(configFile)
		if errMirror != nil {
			t.Fatalf("unexpected error: %s, %v", resultMirror.OutputMsg, errMirror)
		}

		expectedOutMirror := fmt.Sprintf("[WARNING]:-hostAddress has not been provided, populating it with same as hostName %s for the segment with port %d and data_directory %s", valueSegPair[0].Mirror.Hostname, valueSegPair[0].Mirror.Port, valueSegPair[0].Mirror.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", resultMirror.OutputMsg, expectedOutMirror)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	//TODO: FIX this once duplicate key issue is resolved in golang
	// t.Run("when duplicate mirror keys are present", func(t *testing.T) {
	// 	configFile := testutils.GetTempFile(t, "config.json")
	// 	config := GetDefaultConfig(t)

	// 	primarySegs := config.Get("segment-array")
	// 	valueSegPair, ok := primarySegs.([]cli.SegmentPair)
	// 	if !ok {
	// 		t.Fatalf("unexpected data type for segment-array %T", primarySegs)
	// 	}

	// 	// Add duplicate mirror key to the segment pair
	// 	valueSegPair[0].Mirror = &cli.Segment{
	// 		Hostname:      "testhost",
	// 		Address:       "testhost",
	// 		Port:          70010,
	// 		DataDirectory: "/tmp/demo/mirror/gpseg10",
	// 	}

	// 	SetConfigKey(t, configFile, "segment-array", valueSegPair, true)

	// 	result, err := testutils.RunInitCluster(configFile)
	// 	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
	// 		t.Fatalf("got %v, want exit status 1", err)
	// 	}

	// 	expectedOut := "[ERROR]:-duplicate mirror keys are present\n"
	// 	if !strings.Contains(result.OutputMsg, expectedOut) {
	// 		t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
	// 	}
	// })

	t.Run("validate expansion with no coordinator details", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t, true)

		configMap := config.AllSettings()
		delete(configMap, "coordinator")

		encodedConfig, err := json.MarshalIndent(configMap, "", " ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = os.WriteFile(configFile, encodedConfig, 0777)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-no coordinator segment provided in input config file"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion when primary data directory is not provided", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t, true)

		config.Set("primary-data-directories", []string{})
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-primary-data-directories not specified. Please specify primary-data-directories to continue"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion with invalid primary base port is provided", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t, true)

		config.Set("primary-base-port", 0)
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-invalid primary-base-port value provided: 0"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion when empty hostlist is provided", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t, true)

		config.Set("hostlist", []string{})
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-hostlist not specified. Please specify hostlist to continue"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion when empty string is provided for mirror data directory", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		mirrorDataDirectories := config.GetStringSlice("mirror-data-directories")
		if len(mirrorDataDirectories) > 0 {
			mirrorDataDirectories[0] = ""
		}
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-empty mirror-data-directories entry provided, please provide valid directory"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion when invalid mirror base port is provided", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		config.Set("mirror-base-port", 0)
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-invalid mirror-base-port value provided: 0"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion with duplicate ports for primary and mirror", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		config.Set("primary-base-port", 7002)
		config.Set("mirror-base-port", 7002)
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-primary-base-port and mirror-base-port value cannot be same. Please provide different values"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify spread mirroing by providing hosts count less than or equal to primary segment count", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		config.Set("mirroring-type", "spread")
		config.Set("primary-data-directories", append(config.GetStringSlice("primary-data-directories"), "/tmp/additionalprimary"))
		config.Set("mirror-data-directories", append(config.GetStringSlice("mirror-data-directories"), "/tmp/additionalmirror"))
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-To enable spread mirroring, number of hosts should be more than number of primary segments per host"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion with mismatched Number of primary and mirror directories", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		primaryDirs := config.GetStringSlice("primary-data-directories")
		primaryDirs = append(primaryDirs, "/tmp/demo/additionalprimary")
		config.Set("primary-data-directories", primaryDirs)
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write updated config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-number of primary-data-directories should be equal to number of mirror-data-directories"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion with invalid mirror type", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		config.Set("mirroring-type", "test_mirror")
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-invalid mirroring-Type: test_mirror. Valid options are 'group' and 'spread'"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("verify expansion without mirror support", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t, true)
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("Error while intializing cluster: %#v", err)
		}

		expectedWarning := "[WARNING]:-No mirror-data-directories provided. Will create mirror-less cluster"
		if !strings.Contains(result.OutputMsg, expectedWarning) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedWarning)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})
	t.Run("verify expansion with mirror support", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("Error while intializing cluster: %#v", err)
		}

		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("when segment-array is also specified along with the expansion parameter", func(t *testing.T) {
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultExpansionConfig(t)

		segmentArray := []interface{}{map[string]interface{}{}}
		config.Set("segment-array", segmentArray)
		if err := config.WriteConfigAs(configFile); err != nil {
			t.Fatalf("failed to write config to file: %v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := "[ERROR]:-cannot specify segments-array and primary-base-directories together"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("got %q, want %q", result.OutputMsg, expectedOut)
		}

	})
}

func GetDefaultConfig(t *testing.T, mirrorless ...bool) *viper.Viper {
	t.Helper()

	instance := viper.New()
	instance.SetConfigFile("sample_init_config.json")
	instance.SetDefault("common-config", make(map[string]string))
	instance.SetDefault("coordinator-config", make(map[string]string))
	instance.SetDefault("segment-config", make(map[string]string))

	err := instance.ReadInConfig()
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}
	hostList := testutils.GetHostListFromFile(*hostfile)
	coordinatorHost := hostList[0]
	instance.Set("coordinator", cli.Segment{
		Port:          testutils.DEFAULT_COORDINATOR_PORT,
		Hostname:      hostList[0],
		Address:       hostList[0],
		DataDirectory: coordinatorDatadir,
	})

	var segments []cli.SegmentPair
	if len(hostList) == 1 {
		hostList = append(hostList, hostList[0], hostList[0], hostList[0])
	}
	for i := 1; i < len(hostList); i++ {
		hostPrimary := hostList[i]
		if hostPrimary == coordinatorHost {
			hostPrimary = hostList[(i+2)%len(hostList)]
		}
		primary := &cli.Segment{
			Port:          testutils.DEFAULT_COORDINATOR_PORT + i + 1,
			Hostname:      hostPrimary,
			Address:       hostPrimary,
			DataDirectory: filepath.Join("/tmp", "primary", fmt.Sprintf("gpseg%d", i-1)),
		}
		if len(mirrorless) > 0 && mirrorless[0] {
			// Configure only primary segment when mirrorless is true
			segments = append(segments, cli.SegmentPair{
				Primary: primary,
			})
		} else {
			//configure both primary and mirror
			hostMirror := hostList[(i+1)%len(hostList)]
			if hostMirror == coordinatorHost {
				hostMirror = hostList[(i+2)%len(hostList)]
			}
			mirror := &cli.Segment{
				Port:          testutils.DEFAULT_COORDINATOR_PORT + i + 4,
				Hostname:      hostMirror,
				Address:       hostMirror,
				DataDirectory: filepath.Join("/tmp", "mirror", fmt.Sprintf("gpmirror%d", i)),
			}
			segments = append(segments, cli.SegmentPair{
				Primary: primary,
				Mirror:  mirror,
			})
		}
	}
	instance.Set("segment-array", segments)

	return instance
}

func UnsetConfigKey(t *testing.T, filename string, key string, newfile ...bool) {
	t.Helper()

	var config *viper.Viper
	if len(newfile) == 1 && newfile[0] {
		config = GetDefaultConfig(t)
	} else {
		config = viper.New()
		config.SetConfigFile(filename)
		err := config.ReadInConfig()
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
	}

	configMap := config.AllSettings()
	delete(configMap, key)

	encodedConfig, err := json.MarshalIndent(configMap, "", " ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = os.WriteFile(filename, encodedConfig, 0777)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func SetConfigKey(t *testing.T, filename string, key string, value interface{}, newfile ...bool) {
	t.Helper()

	var config *viper.Viper
	if len(newfile) == 1 && newfile[0] {
		config = GetDefaultConfig(t)
	} else {
		config = viper.New()
		config.SetConfigFile(filename)
		err := config.ReadInConfig()
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
	}

	config.Set(key, value)
	err := config.WriteConfigAs(filename)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
func GetDefaultExpansionConfig(t *testing.T, mirrorless ...bool) *viper.Viper {
	t.Helper()
	instance := viper.New()
	instance.SetConfigFile("sample_init_config.json")
	instance.SetDefault("common-config", make(map[string]string))
	instance.SetDefault("coordinator-config", make(map[string]string))
	instance.SetDefault("segment-config", make(map[string]string))

	err := instance.ReadInConfig()
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	hostList := testutils.GetHostListFromFile(*hostfile)

	instance.Set("coordinator", cli.Segment{
		Port:          testutils.DEFAULT_COORDINATOR_PORT,
		Hostname:      hostList[0],
		Address:       hostList[0],
		DataDirectory: coordinatorDatadir,
	})
	//creates mirrorless config when mirrorless param is passed
	if len(mirrorless) == 1 && mirrorless[0] {
		primaryDataDirectories := make([]string, 0)
		for i := 1; i <= 2; i++ {
			primaryDataDirectories = append(primaryDataDirectories, fmt.Sprintf("/tmp/primary%d", i))
		}
		instance.Set("primary-base-port", testutils.DEFAULT_COORDINATOR_PORT+2)
		instance.Set("primary-data-directories", primaryDataDirectories)

	} else { //by default creates both primary and mirror
		primaryDataDirectories := make([]string, 0)
		mirrorDataDirectories := make([]string, 0)

		for i := 1; i <= 2; i++ {
			primaryDataDirectories = append(primaryDataDirectories, fmt.Sprintf("/tmp/primary%d", i))
			mirrorDataDirectories = append(mirrorDataDirectories, fmt.Sprintf("/tmp/mirror%d", i))
		}

		instance.Set("primary-base-port", testutils.DEFAULT_COORDINATOR_PORT+2)
		instance.Set("primary-data-directories", primaryDataDirectories)
		instance.Set("mirror-base-port", testutils.DEFAULT_COORDINATOR_PORT+1002)
		instance.Set("mirroring-type", "group")
		instance.Set("mirror-data-directories", mirrorDataDirectories)
	}
	if len(hostList) == 1 {
		instance.Set("hostlist", hostList)
	} else {
		instance.Set("hostlist", hostList[1:])
	}
	return instance
}
