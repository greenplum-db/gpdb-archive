package init_cluster

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greenplum-db/gpdb/gp/cli"
	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
)

func TestInitCluster(t *testing.T) {
	t.Run("check if the cluster is created successfully and run other utilities to verify - gpstop, gpstart, gpstate, gpcheckcat", func(t *testing.T) {
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
		result, err = testutils.RunGpStatus()
		if err != nil {
			t.Fatalf("Error while getting status of cluster: %#v", err)
		}

		result, err = testutils.RunGpCheckCat()
		if err != nil {
			t.Fatalf("Error while checkcat cluster: %#v", err)
		}
		var expectedOut string
		expectedOut = "Found no catalog issue"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		result, err = testutils.RunGpStop()
		if err != nil {
			t.Fatalf("Error while stopping cluster: %#v", err)
		}
		expectedOut = "[INFO]:-Database successfully shutdown with no errors reported"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		result, err = testutils.RunGpStart()
		if err != nil {
			t.Fatalf("Error while starting cluster: %#v", err)
		}
		expectedOut = "[INFO]:-Database successfully started"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// /* FIXME:concurse is failing to resolve ip to hostname*/
	// t.Run("check if the cluster is created successfully with hba host name set to true and run other utilities to verify - gpstop, gpstart, gpstate, gpcheckcat", func(t *testing.T) {
	// 	if len(hostList) != 1 {
	// 		t.Skip()
	// 	}
	// 	configFile := testutils.GetTempFile(t, "config.json")
	// 	config := GetDefaultConfig(t)

	// 	err := config.WriteConfigAs(configFile)

	// 	if err != nil {
	// 		t.Fatalf("unexpected error: %#v", err)
	// 	}

	// 	SetConfigKey(t, configFile, "hba-hostnames", true, true)

	// 	result, err := testutils.RunInitCluster(configFile)

	// 	if err != nil {
	// 		t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
	// 	}
	// 	result, err = testutils.RunGpStatus()
	// 	if err != nil {
	// 		t.Fatalf("Error while getting status of cluster: %#v", err)
	// 	}
	// 	var expectedOut string
	// 	expectedOut = "[INFO]:-   Coordinator instance                              = Active"
	// 	if !strings.Contains(result.OutputMsg, expectedOut) {
	// 		t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
	// 	}

	// 	result, err = testutils.RunGpCheckCat()
	// 	if err != nil {
	// 		t.Fatalf("Error while checkcat cluster: %#v", err)
	// 	}
	// 	expectedOut = "Found no catalog issue"
	// 	if !strings.Contains(result.OutputMsg, expectedOut) {
	// 		t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
	// 	}

	// 	result, err = testutils.RunGpStop()
	// 	if err != nil {
	// 		t.Fatalf("Error while stopping cluster: %#v", err)
	// 	}
	// 	expectedOut = "[INFO]:-Database successfully shutdown with no errors reported"
	// 	if !strings.Contains(result.OutputMsg, expectedOut) {
	// 		t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
	// 	}

	// 	result, err = testutils.RunGpStart()
	// 	if err != nil {
	// 		t.Fatalf("Error while starting cluster: %#v", err)
	// 	}
	// 	expectedOut = "[INFO]:-Database successfully started"
	// 	if !strings.Contains(result.OutputMsg, expectedOut) {
	// 		t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
	// 	}

	// 	_, err = testutils.DeleteCluster()
	// 	if err != nil {
	// 		t.Fatalf("unexpected error: %v", err)
	// 	}
	// })

	testConfigFileCreation := func(t *testing.T, fileExtension string) {
		configFile := testutils.GetTempFile(t, fmt.Sprintf("config.%s", fileExtension))
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	t.Run("check if the cluster is created successfully by passing config file with yaml extension", func(t *testing.T) {
		testConfigFileCreation(t, "yaml")
	})

	t.Run("check if the cluster is created successfully by passing config file with toml extension", func(t *testing.T) {
		testConfigFileCreation(t, "toml")
	})

	testExpansionConfigFileCreation := func(t *testing.T, fileExtension string) {
		configFile := testutils.GetTempFile(t, fmt.Sprintf("config.%s", fileExtension))
		config := GetDefaultExpansionConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	t.Run("verify expansion by passing config file with yaml extension", func(t *testing.T) {
		testExpansionConfigFileCreation(t, "yaml")
	})

	t.Run("verify expansion by passing config file with toml extension", func(t *testing.T) {
		testExpansionConfigFileCreation(t, "toml")
	})

}

func TestPgHbaConfValidation(t *testing.T) {
	/* FIXME:concurse is failing to resolve ip to hostname*/
	/*t.Run("pghba config file validation when hbahostname is true", func(t *testing.T) {
		var value cli.Segment
		var ok bool
		var valueSeg []cli.Segment
		var okSeg bool
		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		SetConfigKey(t, configFile, "hba-hostnames", true, true)

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}

		filePathCord := filepath.Join(coordinator.(cli.Segment).DataDirectory, "pg_hba.conf")
		hostCord := coordinator.(cli.Segment).Hostname
		cmdStr := "whoami"
		cmd := exec.Command("ssh", hostCord, cmdStr)
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		resultCord := strings.TrimSpace(string(output))
		pgHbaLine := fmt.Sprintf("host\tall\t%s\t%s\ttrust", resultCord, coordinator.(cli.Segment).Hostname)
		cmdStrCord := fmt.Sprintf("/bin/bash -c 'cat %s | grep \"%s\"'", filePathCord, pgHbaLine)
		cmdCord := exec.Command("ssh", hostCord, cmdStrCord)
		_, err = cmdCord.CombinedOutput()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		}

		pgHbaLineSeg := fmt.Sprintf("host\tall\tall\t%s\ttrust", primarySegs.([]cli.Segment)[0].Hostname)
		filePathSeg := filepath.Join(primarySegs.([]cli.Segment)[0].DataDirectory, "pg_hba.conf")
		cmdStr_seg := fmt.Sprintf("/bin/bash -c 'cat %s | grep \"%s\"'", filePathSeg, pgHbaLineSeg)
		hostSeg := primarySegs.([]cli.Segment)[0].Hostname
		cmdSeg := exec.Command("ssh", hostSeg, cmdStr_seg)
		_, err = cmdSeg.CombinedOutput()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})*/

	t.Run("pghba config file validation when hbahostname is false", func(t *testing.T) {
		var value cli.Segment
		var ok bool

		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		SetConfigKey(t, configFile, "hba-hostnames", false, true)

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		coordinator := config.Get("coordinator")
		if value, ok = coordinator.(cli.Segment); !ok {
			t.Fatalf("unexpected data type for coordinator %T", value)
		}

		filePathCord := filepath.Join(coordinator.(cli.Segment).DataDirectory, "pg_hba.conf")
		hostCord := coordinator.(cli.Segment).Hostname
		cmdStr := "whoami"
		cmd := exec.Command("ssh", hostCord, cmdStr)
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		resultCord := strings.TrimSpace(string(output))
		cmdStrCord := "ip -4 addr show | grep inet | grep -v 127.0.0.1/8 | awk '{print $2}'"
		cmdCord := exec.Command("ssh", hostCord, cmdStrCord)
		outputCord, err := cmdCord.Output()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		resultCordValue := string(outputCord)
		firstCordValue := strings.Split(resultCordValue, "\n")[0]
		pgHbaLine := fmt.Sprintf("host\tall\t%s\t%s\ttrust", resultCord, firstCordValue)
		cmdStrCordValue := fmt.Sprintf("/bin/bash -c 'cat %s | grep \"%s\"'", filePathCord, pgHbaLine)
		cmdCordValue := exec.Command("ssh", hostCord, cmdStrCordValue)
		_, err = cmdCordValue.CombinedOutput()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		}

		filePathSeg := filepath.Join(valueSegPair[0].Primary.DataDirectory, "pg_hba.conf")
		hostSegValue := valueSegPair[0].Primary.Hostname
		cmdStrSegValue := "whoami"
		cmdSegvalue := exec.Command("ssh", hostSegValue, cmdStrSegValue)
		outputSeg, errSeg := cmdSegvalue.Output()
		if errSeg != nil {
			t.Fatalf("unexpected error : %v", errSeg)
		}

		resultSeg := strings.TrimSpace(string(outputSeg))
		cmdStrSeg := "ip -4 addr show | grep inet | grep -v 127.0.0.1/8 | awk '{print $2}'"
		cmdSegValueNew := exec.Command("ssh", hostSegValue, cmdStrSeg)
		outputSegNew, err := cmdSegValueNew.Output()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		resultSegValue := string(outputSegNew)
		firstValueNew := strings.Split(resultSegValue, "\n")[0]
		pgHbaLineNew := fmt.Sprintf("host\tall\t%s\t%s\ttrust", resultSeg, firstValueNew)
		cmdStrSegNew := fmt.Sprintf("/bin/bash -c 'cat %s | grep \"%s\"'", filePathSeg, pgHbaLineNew)
		cmdSegNew := exec.Command("ssh", hostSegValue, cmdStrSegNew)
		_, err = cmdSegNew.CombinedOutput()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validate pg_hba.conf mirror replication entry in primary segment", func(t *testing.T) {
		var ok bool

		configFile := testutils.GetTempFile(t, "config.json")
		config := GetDefaultConfig(t)

		err := config.WriteConfigAs(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		SetConfigKey(t, configFile, "hba-hostnames", false, true)

		result, err := testutils.RunInitCluster(configFile)
		if err != nil {
			t.Fatalf("unexpected error: %s, %v", result.OutputMsg, err)
		}

		primarySegs := config.Get("segment-array")
		valueSegPair, ok := primarySegs.([]cli.SegmentPair)

		if !ok {
			t.Fatalf("unexpected data type for segment-array %T", primarySegs)
		}

		primaryHbaConfFilePath := filepath.Join(valueSegPair[0].Primary.DataDirectory, "pg_hba.conf")

		//fetches the user name of the mirror host
		mirrorHostName := valueSegPair[0].Mirror.Hostname
		UserNameCmd := "whoami"
		cmd := exec.Command("ssh", mirrorHostName, UserNameCmd)
		outputSeg, errSeg := cmd.Output()
		if errSeg != nil {
			t.Fatalf("unexpected error : %v", errSeg)
		}
		mirrorUserName := strings.TrimSpace(string(outputSeg))

		//fetches the IP address of the mirror host
		ipAddressCmd := "ip -4 addr show | grep inet | grep -v 127.0.0.1/8 | awk '{print $2}'"
		cmdObj := exec.Command("ssh", mirrorHostName, ipAddressCmd)
		outputCmd, err := cmdObj.Output()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}
		mirrorIPAddress := string(outputCmd)
		mirrorIPAddressString := strings.Split(mirrorIPAddress, "\n")[0]

		//validates mirror replication entry in primary host
		primaryrHostName := valueSegPair[0].Primary.Hostname
		replicationEntryCmdStr := fmt.Sprintf("host\treplication\t%s\t%s\ttrust", mirrorUserName, mirrorIPAddressString)
		replicationEntryCmdObj := fmt.Sprintf("/bin/bash -c 'cat %s | grep \"%s\"'", primaryHbaConfFilePath, replicationEntryCmdStr)
		replicationEntryResult := exec.Command("ssh", primaryrHostName, replicationEntryCmdObj)
		_, err = replicationEntryResult.CombinedOutput()
		if err != nil {
			t.Fatalf("unexpected error : %v", err)
		}

		_, err = testutils.DeleteCluster()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

}
