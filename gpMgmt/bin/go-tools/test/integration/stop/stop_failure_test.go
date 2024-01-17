package stop

import (
	"os"
	"strings"
	"testing"

	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
)

func TestStopFailsWithoutSvcRunning(t *testing.T) {
	t.Run("stop agents fails when hub is not running", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")
		_, _ = testutils.RunStop("hub")

		expectedOut := []string{
			"error stopping agent service", "could not connect to hub",
		}

		result, err := testutils.RunStop("agents")
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

	t.Run("stop services fails when services are not running", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		expectedOut := "could not connect to hub"

		result, err := testutils.RunStop("services")
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

	t.Run("stop hub fails when hub is not running", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)

		expectedOut := "could not connect to hub"
		result, err := testutils.RunStop("hub")
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

	t.Run("stop agents fails when services are not running", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)

		expectedOut := "could not connect to hub"
		result, err := testutils.RunStop("agents")
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

func TestStopFailureWithoutConfig(t *testing.T) {
	TestCases := []struct {
		name        string
		cliParams   []string
		expectedOut []string
	}{
		{
			name: "stop services fails when service configuration file is not present",
			cliParams: []string{
				"services",
			},
			expectedOut: []string{
				"could not open config file", "no such file or directory",
			},
		},
		{
			name: "stop hub fails when service configuration file is not present",
			cliParams: []string{
				"hub",
			},
			expectedOut: []string{
				"could not open config file", "no such file or directory",
			},
		},
		{
			name: "stop agents fails when service configuration file is not present",
			cliParams: []string{
				"agents",
			},
			expectedOut: []string{
				"could not open config file", "no such file or directory",
			},
		},
	}
	for _, tc := range TestCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _ = testutils.RunStart("services")
			_ = testutils.CopyFile(testutils.DefaultConfigurationFile, "/tmp/config.conf")
			_ = os.RemoveAll(testutils.DefaultConfigurationFile)

			result, err := testutils.RunStop(tc.cliParams...)
			if err == nil {
				t.Errorf("\nExpected error Got: %#v", err)
			}
			if result.ExitCode != testutils.ExitCode1 {
				t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
			}
			for _, item := range tc.expectedOut {
				if !strings.Contains(result.OutputMsg, item) {
					t.Errorf("\nExpected string: %#v \nNot found in: %#v", item, result.OutputMsg)
				}
			}
			_, _ = testutils.RunStop("services", "--config-file", "/tmp/config.conf")
		})
	}
}

func TestStopFailsWithoutCertificates(t *testing.T) {
	TestCases := []struct {
		name        string
		cliParams   []string
		expectedOut []string
	}{
		{
			name: "stop services fails when certificates are not present",
			cliParams: []string{
				"services", "--config-file", configCopy,
			},
			expectedOut: []string{
				"could not connect to hub",
			},
		},
		{
			name: "stop hub fails when certificates are not present",
			cliParams: []string{
				"hub", "--config-file", configCopy,
			},
			expectedOut: []string{
				"could not connect to hub",
			},
		},
		{
			name: "stop agents fails when certificates are not present",
			cliParams: []string{
				"agents", "--config-file", configCopy,
			},
			expectedOut: []string{
				"error stopping agent service",
			},
		},
	}
	for _, tc := range TestCases {
		t.Run(tc.name, func(t *testing.T) {
			testutils.InitService(*hostfile, testutils.CertificateParams)
			_, _ = testutils.RunStart("services")
			_ = testutils.CpCfgWithoutCertificates(configCopy)

			result, err := testutils.RunStop(tc.cliParams...)
			if err == nil {
				t.Errorf("\nExpected error Got: %#v", err)
			}
			if result.ExitCode != testutils.ExitCode1 {
				t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
			}
			for _, item := range tc.expectedOut {
				if !strings.Contains(result.OutputMsg, item) {
					t.Errorf("\nExpected string: %#v \nNot found in: %#v", item, result.OutputMsg)
				}
			}
			_, _ = testutils.RunStop("services")
		})
	}
}

func TestStopFailsWithInvalidOptions(t *testing.T) {
	TestCases := []struct {
		name        string
		cliParams   []string
		expectedOut []string
	}{
		{
			name: "stop services with no value for --config-file will fail",
			cliParams: []string{
				"services", "--config-file",
			},
			expectedOut: []string{
				"flag needs an argument: --config-file",
			},
		},
		{
			name: "stop services with non-existing file for --config-file will fail",
			cliParams: []string{
				"services", "--config-file", "file",
			},
			expectedOut: []string{
				"no such file or directory",
			},
		},
		{
			name: "stop services with empty string for --config-file will fail",
			cliParams: []string{
				"services", "--config-file", "",
			},
			expectedOut: []string{
				"no such file or directory",
			},
		},
	}

	for _, tc := range TestCases {
		t.Run(tc.name, func(t *testing.T) {
			testutils.InitService(*hostfile, testutils.CertificateParams)
			_, _ = testutils.RunStart("services")

			result, err := testutils.RunStop(tc.cliParams...)
			if err == nil {
				t.Errorf("\nExpected error Got: %#v", err)
			}
			if result.ExitCode != testutils.ExitCode1 {
				t.Errorf("\nExpected: %#v \nGot: %v", testutils.ExitCode1, result.ExitCode)
			}
			for _, item := range tc.expectedOut {
				if !strings.Contains(result.OutputMsg, item) {
					t.Errorf("\nExpected string: %#v \nNot found in: %#v", item, result.OutputMsg)
				}
			}
			_, _ = testutils.RunStop("services")
		})
	}
}
