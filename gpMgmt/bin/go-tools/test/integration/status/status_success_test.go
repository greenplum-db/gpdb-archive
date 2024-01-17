package status

import (
	"github.com/greenplum-db/gpdb/gp/hub"
	"github.com/greenplum-db/gpdb/gp/test/integration/testutils"
	"strings"
	"testing"
)

var gpCfg hub.Config

func TestStatusSuccess(t *testing.T) {
	var StatusSuccessTestCases = []struct {
		name        string
		cliParams   []string
		expectedOut []string
		serviceName []string
	}{
		{
			name: "status services shows status of hub and agents",
			cliParams: []string{
				"services",
			},
			expectedOut: []string{
				"ROLE", "HOST", "STATUS", "PID", "UPTIME",
				"Hub", "running",
				"Agent", "running",
			},
			serviceName: []string{
				"gp_hub",
				"gp_agent",
			},
		},
		{
			name: "status hub shows status of hub",
			cliParams: []string{
				"hub",
			},
			expectedOut: []string{
				"ROLE", "HOST", "STATUS", "PID", "UPTIME",
				"Hub", "running",
			},
			serviceName: []string{
				"gp_hub",
			},
		},
		{
			name: "status agents shows status of agents",
			cliParams: []string{
				"agents",
			},
			expectedOut: []string{
				"ROLE", "HOST", "STATUS", "PID", "UPTIME",
				"Agent", "running",
			},
			serviceName: []string{
				"gp_agent",
			},
		},
		{
			name: "status services with --verbose cli param",
			cliParams: []string{
				"services", "--verbose",
			},
			expectedOut: []string{
				"ROLE", "HOST", "STATUS", "PID", "UPTIME",
				"Hub", "running",
				"Agent", "running",
			},
			serviceName: []string{
				"gp_hub",
				"gp_agent",
			},
		},
		{
			name: "status hub with --verbose cli param",
			cliParams: []string{
				"hub", "--verbose",
			},
			expectedOut: []string{
				"ROLE", "HOST", "STATUS", "PID", "UPTIME",
				"Hub", "running",
			},
			serviceName: []string{
				"gp_hub",
			},
		},
		{
			name: "status agents with --verbose cli param",
			cliParams: []string{
				"agents", "--verbose",
			},
			expectedOut: []string{
				"ROLE", "HOST", "STATUS", "PID", "UPTIME",
				"Agent", "running",
			},
			serviceName: []string{
				"gp_agent",
			},
		},
	}

	for _, tc := range StatusSuccessTestCases {
		t.Run(tc.name, func(t *testing.T) {
			testutils.InitService(*hostfile, testutils.CertificateParams)
			_, _ = testutils.RunStart("services")
			gpCfg = testutils.ParseConfig(testutils.DefaultConfigurationFile)

			// Running the gp status command
			result, err := testutils.RunStatus(tc.cliParams...)
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

			// verify the pid in status is listening on correct port
			statusMap := testutils.ExtractStatusData(result.OutputMsg)
			for _, svc := range tc.serviceName {
				listeningPort := gpCfg.Port
				hostPidMap := statusMap["Hub"]
				if svc == "gp_agent" {
					listeningPort = gpCfg.AgentPort
					hostPidMap = statusMap["Agent"]
				}
				for host, pid := range hostPidMap {
					testutils.VerifyServicePIDOnPort(t, pid, listeningPort, host)
				}
			}
			_, _ = testutils.RunStop("services")
		})
	}
}

func TestStatusSuccessWithoutDefaultService(t *testing.T) {
	t.Run("status services when gp installed with --service-name param", func(t *testing.T) {
		params := append(testutils.CertificateParams, []string{"--service-name", "dummySvc"}...)
		testutils.InitService(*hostfile, params)
		_, _ = testutils.RunStart("services")

		cliParams := []string{
			"services", "--verbose",
		}
		expectedOut := []string{
			"ROLE", "HOST", "STATUS", "PID", "UPTIME",
			"Hub", "running",
			"Agent", "running",
		}

		gpCfg = testutils.ParseConfig(testutils.DefaultConfigurationFile)

		result, err := testutils.RunStatus(cliParams...)
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

		// verify the pid in status is listening on correct port
		statusMap := testutils.ExtractStatusData(result.OutputMsg)
		for _, svc := range []string{"dummySvc_hub", "dummySvc_agent"} {
			listeningPort := gpCfg.Port
			hostPidMap := statusMap["Hub"]
			if svc == "dummySvc_agent" {
				listeningPort = gpCfg.AgentPort
				hostPidMap = statusMap["Agent"]
			}
			for host, pid := range hostPidMap {
				testutils.VerifyServicePIDOnPort(t, pid, listeningPort, host)
			}
		}
		_, _ = testutils.RunStop("services")
	})

	t.Run("status hub shows status of hub when it is not running", func(t *testing.T) {
		testutils.InitService(*hostfile, testutils.CertificateParams)

		expectedOut := []string{
			"ROLE", "HOST", "STATUS", "PID", "UPTIME",
			"Hub", "not running",
		}

		// Running the gp status command for hub
		result, err := testutils.RunStatus("hub")
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
	})
}

func TestStatusSuccessHelp(t *testing.T) {
	TestCases := []struct {
		name        string
		cliParams   []string
		expectedOut []string
	}{
		{
			name: "status command without params shows help",
			expectedOut: append([]string{
				"Display status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status command with invalid param shows help",
			cliParams: []string{
				"invalid",
			},
			expectedOut: append([]string{
				"Display status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status command with --help shows help",
			cliParams: []string{
				"--help",
			},
			expectedOut: append([]string{
				"Display status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status command with -h shows help",
			cliParams: []string{
				"-h",
			},
			expectedOut: append([]string{
				"Display status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status hub command with --help shows help",
			cliParams: []string{
				"hub", "--help",
			},
			expectedOut: append([]string{
				"Display hub status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status hub command with -h shows help",
			cliParams: []string{
				"hub", "-h",
			},
			expectedOut: append([]string{
				"Display hub status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status agents command with --help shows help",
			cliParams: []string{
				"agents", "--help",
			},
			expectedOut: append([]string{
				"Display agents status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status agents command with -h shows help",
			cliParams: []string{
				"agents", "-h",
			},
			expectedOut: append([]string{
				"Display agents status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status services command with --help shows help",
			cliParams: []string{
				"services", "--help",
			},
			expectedOut: append([]string{
				"Display Hub and Agent services status",
			}, testutils.CommonHelpText...),
		},
		{
			name: "status services command with -h shows help",
			cliParams: []string{
				"services", "-h",
			},
			expectedOut: append([]string{
				"Display Hub and Agent services status",
			}, testutils.CommonHelpText...),
		},
	}

	for _, tc := range TestCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutils.RunStatus(tc.cliParams...)
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
