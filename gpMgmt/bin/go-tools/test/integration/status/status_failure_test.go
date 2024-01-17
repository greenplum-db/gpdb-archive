package status

import (
	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
	"os"
	"strings"
	"testing"
)

func TestStatusFailures(t *testing.T) {
	t.Run("checking service status without configuration file will fail", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")
		_ = testutils.CopyFile(testutils.DefaultConfigurationFile, "/tmp/config.conf")
		_ = os.RemoveAll(testutils.DefaultConfigurationFile)

		expectedOut := "could not open config file"

		result, err := testutils.RunStatus("services")
		if err == nil {
			t.Errorf("\nExpected error Got: %#v", err)
		}
		if result.ExitCode != testutils.ExitCode1 {
			t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		_, _ = testutils.RunStop("services", "--config-file", "/tmp/config.conf")
	})

	t.Run("checking status of agents will fail if hub is not running", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)

		expectedOut := "could not connect to hub"
		result, err := testutils.RunStatus("agents")
		if err == nil {
			t.Errorf("\nExpected error Got: %#v", err)
		}
		if result.ExitCode != testutils.ExitCode1 {
			t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}
	})

	t.Run("checking status of services after stopping hub will fail", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)

		expectedOut := []string{
			"Hub", "not running", "0",
			"could not connect to hub",
		}

		result, err := testutils.RunStatus("services")
		if err == nil {
			t.Errorf("\nExpected error Got: %#v", err)
		}
		if result.ExitCode != testutils.ExitCode1 {
			t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
		}
		for _, item := range expectedOut {
			if !strings.Contains(result.OutputMsg, item) {
				t.Errorf("\nExpected string: %#v \nNot found in: %#v", item, result.OutputMsg)
			}
		}
	})

	t.Run("checking status of agents without certificates", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")
		_ = testutils.CpCfgWithoutCertificates(configCopy)

		cliParams := []string{
			"agents", "--config-file", configCopy,
		}
		expectedOut := "error while loading server certificate"

		result, err := testutils.RunStatus(cliParams...)
		if err == nil {
			t.Errorf("\nExpected error Got: %#v", err)
		}
		if result.ExitCode != testutils.ExitCode1 {
			t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		_, _ = testutils.RunStop("services")
	})

	t.Run("checking status of services without certificates", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")
		_ = testutils.CpCfgWithoutCertificates(configCopy)

		cliParams := []string{
			"services", "--config-file", configCopy,
		}
		expectedOut := "error while loading server certificate"

		result, err := testutils.RunStatus(cliParams...)
		if err == nil {
			t.Errorf("\nExpected error Got: %#v", err)
		}
		if result.ExitCode != testutils.ExitCode1 {
			t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		_, _ = testutils.RunStop("services")
	})

	t.Run("checking service status with no value for --config-file will fail", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")

		cliParams := []string{
			"services", "--config-file",
		}
		expectedOut := "flag needs an argument: --config-file"

		result, err := testutils.RunStatus(cliParams...)
		if err == nil {
			t.Errorf("\nExpected error Got: %#v", err)
		}
		if result.ExitCode != testutils.ExitCode1 {
			t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		_, _ = testutils.RunStop("services")
	})

	t.Run("checking service status with non-existing file for --config-file will fail", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")

		cliParams := []string{
			"services", "--config-file", "file",
		}
		expectedOut := "no such file or directory"

		result, err := testutils.RunStatus(cliParams...)
		if err == nil {
			t.Errorf("\nExpected error Got: %#v", err)
		}
		if result.ExitCode != testutils.ExitCode1 {
			t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		_, _ = testutils.RunStop("services")
	})

	t.Run("checking service status with empty string for --config-file will fail", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")

		cliParams := []string{
			"services", "--config-file", "",
		}
		expectedOut := "no such file or directory"

		result, err := testutils.RunStatus(cliParams...)
		if err == nil {
			t.Errorf("\nExpected error Got: %#v", err)
		}
		if result.ExitCode != testutils.ExitCode1 {
			t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		_, _ = testutils.RunStop("services")
	})
}
