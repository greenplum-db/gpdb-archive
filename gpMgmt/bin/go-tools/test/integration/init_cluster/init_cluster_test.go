package init_cluster

import (
	"fmt"
	"strings"
	"testing"

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
		var expectedOut string
		expectedOut = "[INFO]:-   Coordinator instance                              = Active"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		result, err = testutils.RunGpCheckCat()
		if err != nil {
			t.Fatalf("Error while checkcat cluster: %#v", err)
		}
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

	t.Run("check if the cluster is created successfully with hba host name set to true and run other utilities to verify - gpstop, gpstart, gpstate, gpcheckcat", func(t *testing.T) {
		if len(hostList) != 1 {
			t.Skip()
		}
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
		result, err = testutils.RunGpStatus()
		if err != nil {
			t.Fatalf("Error while getting status of cluster: %#v", err)
		}
		var expectedOut string
		expectedOut = "[INFO]:-   Coordinator instance                              = Active"
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Fatalf("got %q, want %q", result.OutputMsg, expectedOut)
		}

		result, err = testutils.RunGpCheckCat()
		if err != nil {
			t.Fatalf("Error while checkcat cluster: %#v", err)
		}
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
}
