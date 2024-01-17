package start

import (
	"os"
	"strings"
	"testing"

	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
)

func TestStartFailWithoutConfig(t *testing.T) {
	t.Run("starting services without configuration file will fail", func(t *testing.T) {
		_ = os.RemoveAll(testutils.DefaultConfigurationFile)
		expectedOut := []string{
			"could not open config file",
			"no such file or directory",
		}
		// start services
		result, err := testutils.RunStart("services")
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

	t.Run("starting hub without service file", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		testutils.DisableandDeleteServiceFiles(p)

		expectedOut := "failed to start hub service"

		// start hub
		result, err := testutils.RunStart("hub")
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

	t.Run("starting hub without certificates", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_ = testutils.CpCfgWithoutCertificates(configCopy)

		expectedOut := "error while loading server certificate"

		result, err := testutils.RunStart("hub", "--config-file", configCopy)
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

	t.Run("starting agents without certificates", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_ = testutils.CpCfgWithoutCertificates(configCopy)

		expectedOut := "error while loading server certificate"

		// start agents
		result, err := testutils.RunStart("agents", "--config-file", configCopy)
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

	t.Run("starting services without certificates", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_ = testutils.CpCfgWithoutCertificates(configCopy)

		expectedOut := "error while loading server certificate"
		// start agents
		result, err := testutils.RunStart("services", "--config-file", configCopy)
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
}

func TestStartGlobalFlagsFailures(t *testing.T) {
	failTestCases := []struct {
		name        string
		cliParams   []string
		expectedOut string
	}{
		{
			name: "starting agents without starting hub will fail",
			cliParams: []string{
				"agents",
			},
			expectedOut: "could not connect to hub",
		},
		{
			name: "starting services with no value for --config-file will fail",
			cliParams: []string{
				"services", "--config-file",
			},
			expectedOut: "flag needs an argument: --config-file",
		},
		{
			name: "starting services with non-existing file for --config-file will fail",
			cliParams: []string{
				"services", "--config-file", "file",
			},
			expectedOut: "no such file or directory",
		},
		{
			name: "starting services with empty string for --config-file will fail",
			cliParams: []string{
				"services", "--config-file", "",
			},
			expectedOut: "no such file or directory",
		},
	}

	for _, tc := range failTestCases {
		t.Run(tc.name, func(t *testing.T) {
			testutils.DisableandDeleteServiceFiles(p)
			testutils.InitService(*hostfile, testutils.CertificateParams)

			result, err := testutils.RunStart(tc.cliParams...)
			if err == nil {
				t.Errorf("\nExpected error Got: %#v", err)
			}
			if result.ExitCode != testutils.ExitCode1 {
				t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
			}
			if !strings.Contains(result.OutputMsg, tc.expectedOut) {
				t.Errorf("\nExpected string: %#v \nNot found in: %#v", tc.expectedOut, result.OutputMsg)
			}

		})
		_, _ = testutils.RunStop("services")
	}
}
