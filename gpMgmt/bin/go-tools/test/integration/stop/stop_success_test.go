package stop

import (
	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
	"github.com/greenplum-db/gpdb/gp/utils"
	"strings"
	"testing"
)

func TestStopSuccess(t *testing.T) {
	hosts := testutils.GetHostListFromFile(*hostfile)

	t.Run("stop services successfully", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")

		expectedOut := []string{
			"Agents stopped successfully",
			"Hub stopped successfully",
		}

		// Running the gp stop command for services
		result, err := testutils.RunStop("services")
		// check for command result
		if err != nil {
			t.Errorf("\nUnexpected error: %#v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("\nExpected: %v \nGot: %v", 0, result.ExitCode)
		}
		for _, item := range expectedOut {
			if !strings.Contains(result.OutputMsg, item) {
				t.Errorf("\nExpected string: %#v \nNot found in: %#v", item, result.OutputMsg)
			}
		}

		// check if service is not running
		for _, svc := range []string{"gp_hub", "gp_agent"} {
			hostList := hosts[:1]
			if svc == "gp_agent" {
				hostList = hosts
			}
			for _, host := range hostList {
				status, _ := testutils.GetSvcStatusOnHost(p.(utils.GpPlatform), svc, host)
				testutils.VerifySvcNotRunning(t, status.OutputMsg)
			}
		}
	})

	t.Run("stop hub successfully", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("hub")

		expectedOut := "Hub stopped successfully"

		// Running the gp stop command for hub
		result, err := testutils.RunStop("hub")
		// check for command result
		if err != nil {
			t.Errorf("\nUnexpected error: %#v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("\nExpected: %v \nGot: %v", 0, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		// check if service is not running
		status, _ := testutils.GetSvcStatusOnHost(p.(utils.GpPlatform), "gp_hub", hosts[0])
		testutils.VerifySvcNotRunning(t, status.OutputMsg)
	})

	t.Run("stop agents successfully", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")

		expectedOut := "Agents stopped successfully"

		// Running the gp stop command for agents
		result, err := testutils.RunStop("agents")
		// check for command result
		if err != nil {
			t.Errorf("\nUnexpected error: %#v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("\nExpected: %v \nGot: %v", 0, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		// check if service is not running
		for _, host := range hosts {
			status, _ := testutils.GetSvcStatusOnHost(p.(utils.GpPlatform), "gp_agent", host)
			testutils.VerifySvcNotRunning(t, status.OutputMsg)
		}

		_, _ = testutils.RunStop("hub")
	})

	t.Run("stop services command with --verbose shows status details", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")

		cliParams := []string{
			"services", "--verbose",
		}
		expectedOut := []string{
			"Agents stopped successfull",
			"Hub stopped successfully",
		}

		result, err := testutils.RunStop(cliParams...)
		// check for command result
		if err != nil {
			t.Errorf("\nUnexpected error: %#v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("\nExpected: %v \nGot: %v", 0, result.ExitCode)
		}
		for _, item := range expectedOut {
			if !strings.Contains(result.OutputMsg, item) {
				t.Errorf("\nExpected string: %#v \nNot found in: %#v", item, result.OutputMsg)
			}
		}

		// check if service is not running
		for _, svc := range []string{"gp_hub", "gp_agent"} {
			hostList := hosts[:1]
			if svc == "gp_agent" {
				hostList = hosts
			}
			for _, host := range hostList {
				status, _ := testutils.GetSvcStatusOnHost(p.(utils.GpPlatform), svc, host)
				testutils.VerifySvcNotRunning(t, status.OutputMsg)
			}
		}
	})

	t.Run("stop hub command with --verbose shows status details", func(t *testing.T) {
		testutils.InitService(testutils.DefaultHost, testutils.CertificateParams)
		_, _ = testutils.RunStart("hub")

		cliParams := []string{
			"hub", "--verbose",
		}
		expectedOut := []string{
			"Hub stopped successfully",
			"ROLE", "HOST", "STATUS", "PID", "UPTIME",
			"Hub", "not running", "0",
		}

		result, err := testutils.RunStop(cliParams...)
		// check for command result
		if err != nil {
			t.Errorf("\nUnexpected error: %#v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("\nExpected: %v \nGot: %v", 0, result.ExitCode)
		}
		for _, item := range expectedOut {
			if !strings.Contains(result.OutputMsg, item) {
				t.Errorf("\nExpected string: %#v \nNot found in: %#v", item, result.OutputMsg)
			}
		}

		// check if service is not running
		status, _ := testutils.GetSvcStatusOnHost(p.(utils.GpPlatform), "gp_hub", hosts[0])
		testutils.VerifySvcNotRunning(t, status.OutputMsg)
	})

	t.Run("stop agents command with --verbose", func(t *testing.T) {
		testutils.InitService(testutils.DefaultHost, testutils.CertificateParams)
		_, _ = testutils.RunStart("services")

		cliParams := []string{
			"agents", "--verbose",
		}
		expectedOut := "Agents stopped successfully"
		
		result, err := testutils.RunStop(cliParams...)
		// check for command result
		if err != nil {
			t.Errorf("\nUnexpected error: %#v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("\nExpected: %v \nGot: %v", 0, result.ExitCode)
		}
		if !strings.Contains(result.OutputMsg, expectedOut) {
			t.Errorf("\nExpected string: %#v \nNot found in: %#v", expectedOut, result.OutputMsg)
		}

		// check if service is not running
		for _, host := range hosts {
			status, _ := testutils.GetSvcStatusOnHost(p.(utils.GpPlatform), "gp_agent", host)
			testutils.VerifySvcNotRunning(t, status.OutputMsg)
		}

		_, _ = testutils.RunStop("hub")
	})
}

func TestStopSuccessHelp(t *testing.T) {
	TestCases := []struct {
		name        string
		cliParams   []string
		expectedOut []string
	}{
		{
			name: "stop command without params shows help",
			expectedOut: append([]string{
				"Stop processes",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop command with invalid param shows help",
			cliParams: []string{
				"invalid",
			},
			expectedOut: append([]string{
				"Stop processes",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop command with --help shows help",
			cliParams: []string{
				"--help",
			},
			expectedOut: append([]string{
				"Stop processes",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop command with -h shows help",
			cliParams: []string{
				"-h",
			},
			expectedOut: append([]string{
				"Stop processes",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop hub command with --help shows help",
			cliParams: []string{
				"hub", "--help",
			},
			expectedOut: append([]string{
				"Stop hub",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop hub command with -h shows help",
			cliParams: []string{
				"hub", "-h",
			},
			expectedOut: append([]string{
				"Stop hub",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop agents command with --help shows help",
			cliParams: []string{
				"agents", "--help",
			},
			expectedOut: append([]string{
				"Stop agents",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop agents command with -h shows help",
			cliParams: []string{
				"agents", "-h",
			},
			expectedOut: append([]string{
				"Stop agents",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop services command with --help shows help",
			cliParams: []string{
				"services", "--help",
			},
			expectedOut: append([]string{
				"Stop hub and agent services",
			}, testutils.CommonHelpText...),
		},
		{
			name: "stop services command with -h shows help",
			cliParams: []string{
				"services", "-h",
			},
			expectedOut: append([]string{
				"Stop hub and agent services",
			}, testutils.CommonHelpText...),
		},
	}
	for _, tc := range TestCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutils.RunStop(tc.cliParams...)
			// check for command result
			if err != nil {
				t.Errorf("\nUnexpected error: %#v", err)
			}
			if result.ExitCode != 0 {
				t.Errorf("\nExpected: %v \nGot: %v", 0, result.ExitCode)
			}
			for _, item := range tc.expectedOut {
				if !strings.Contains(result.OutputMsg, item) {
					t.Errorf("\nExpected string: %#v \nNot found in: %#v", item, result.OutputMsg)
				}
			}
		})
	}
}
