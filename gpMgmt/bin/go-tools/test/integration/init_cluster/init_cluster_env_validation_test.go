package init_cluster

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/greenplum-db/gpdb/gp/cli"
	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
)

func TestEnvValidation(t *testing.T) {
	t.Run("when the given data directory is not empty", func(t *testing.T) {
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
			t.Fatalf("unexpected data type for coordinator %T", coordinator)
		}

		err = os.MkdirAll(value.DataDirectory, 0777)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		_, err = os.Create(filepath.Join(value.DataDirectory, "abc.txt"))
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		defer os.RemoveAll(value.DataDirectory)

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-validating hosts: host: %s, directory not empty:[%s]\n", value.Hostname, value.DataDirectory)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when the given port is already in use", func(t *testing.T) {
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

		lis, err := net.Listen("tcp", net.JoinHostPort(value.Address, strconv.Itoa(value.Port)))
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
		defer lis.Close()

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("ERROR]:-validating hosts: host: %s, ports already in use: [%d], check if cluster already running", value.Hostname, value.Port)
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when the initdb does not have appropriate permission", func(t *testing.T) {
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

		gpHome := os.Getenv("GPHOME")
		if gpHome == "" {
			t.Fatal("GPHOME environment variable not set")
		}

		initdbFilePath := filepath.Join(gpHome, "bin", "initdb")

		if err := os.Chmod(initdbFilePath, 0444); err != nil {
			t.Fatalf("unexpected error during changing initdb file permission: %#v", err)
		}
		defer func() {
			if err := os.Chmod(initdbFilePath, 0755); err != nil {
				t.Fatalf("unexpected error during changing initdb file permission: %#v", err)
			}
		}()

		result, err := testutils.RunInitCluster(configFile)
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		expectedOut := fmt.Sprintf("[ERROR]:-validating hosts: host: %s, file %s does not have execute permissions", value.Hostname, initdbFilePath)

		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}
	})

	t.Run("when data directory is not empty and --force is given for gp init command", func(t *testing.T) {
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

		err = os.MkdirAll(value.DataDirectory, 0777)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		_, err = os.Create(filepath.Join(value.DataDirectory, "abc.txt"))
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster("--force", configFile)

		if err != nil {
			t.Fatalf("Error while intializing cluster: %#v", err)
		}
		expectedOut := "[INFO]:-Cluster initialized successfully"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		//FIXME - with force flag is triggered immediately again initilization fails, below part of code should be enabled once bug is fixed
		// initResult, err := testutils.RunInitCluster("--force", configFile)

		// if err != nil {
		// 	t.Errorf("Error while intializing cluster: %#v", err)
		// }

		// clusterExpectedOut := "[INFO]:-Cluster initialized successfully"
		// if !strings.Contains(result.OutputMsg, expectedOut) {
		// 	t.Errorf("got %q, want %q", initResult.OutputMsg, clusterExpectedOut)
		// }

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("check if the coordinator is stopped whenever an error occurs", func(t *testing.T) {
		var valueSegPair []cli.SegmentPair
		var okSeg bool
		var value cli.Segment
		var ok bool

		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		primarySegs := config.Get("segment-array")
		if valueSegPair, okSeg = primarySegs.([]cli.SegmentPair); !okSeg {
			t.Fatalf("unexpected data type for segment-array %T", valueSegPair)
		}

		pSegPath := filepath.Join(valueSegPair[0].Primary.DataDirectory, "postgresql.conf")
		host := valueSegPair[0].Primary.Hostname
		coordinator := config.Get("coordinator")

		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}
		coordinatorDD := value.DataDirectory

		dirDeleted := make(chan bool)
		go func() {
			for {
				select {
				case <-dirDeleted:
					return
				default:
					cmdStr := fmt.Sprintf("if [ -f %s ]; then echo 'exists'; fi", pSegPath)
					cmd := exec.Command("ssh", host, cmdStr)
					output, err := cmd.Output()
					if err != nil {
						t.Errorf("unexpected error: %#v", err)
					}

					if strings.TrimSpace(string(output)) == "exists" {
						cmdStr := fmt.Sprintf("rm -rf %s", pSegPath)
						cmd := exec.Command("ssh", host, cmdStr)
						_, err := cmd.CombinedOutput()
						if err != nil {
							t.Errorf("unexpected error: %#v", err)
						}
						dirDeleted <- true
					}
					time.Sleep(1 * time.Second)
				}
			}
		}()

		_, initErr := testutils.RunInitCluster(configFile)
		if e, ok := initErr.(*exec.ExitError); !ok || e.ExitCode() != 1 {
			t.Fatalf("got %v, want exit status 1", err)
		}

		// check if the coordinator is stopped
		cmd := exec.Command("pg_ctl", "status", "-D", coordinatorDD)
		cmdOuput, err := cmd.CombinedOutput()
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 3 {
			t.Fatalf("got %s, %v, want exit status 3", cmdOuput, err)
		}

		expectedOut := "pg_ctl: no server running"
		if !strings.Contains(string(cmdOuput), expectedOut) {
			t.Errorf("got %q, want %q", cmdOuput, expectedOut)
		}

		_, initClusterErr := testutils.RunInitCluster("--force", configFile)
		if initClusterErr != nil {
			t.Fatalf("Error while intializing cluster: %v", initClusterErr)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
