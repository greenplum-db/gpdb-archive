package testutils

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func FilesExistsOnAgents(t *testing.T, file string, hosts []string) bool {
	cmdStr := fmt.Sprintf("/bin/bash -c 'test -e %s && echo $?'", file)
	for _, host := range hosts {
		cmd := exec.Command("ssh", host, cmdStr)
		out, _ := cmd.CombinedOutput()
		if strings.TrimSpace(string(out)) != "0" {
			t.Errorf("File %s not found on %s", file, host)
		}
	}

	return true
}

func FilesExistOnHub(t *testing.T, files ...string) bool {
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			t.Errorf("File %s not found", file)
		}
	}
	return true
}

func VerifyServicePIDOnPort(t *testing.T, PidStatus string, port int, host string) bool {
	var pid string
	if _, err := strconv.Atoi(PidStatus); err != nil {
		pid = extractPID(PidStatus)
	} else {
		pid = PidStatus
	}

	listeningPid := GetListeningProcess(port, host)
	if pid != listeningPid {
		t.Errorf("pid %s in service status not matching with pid(%s) listening on port %d ", pid, listeningPid, port)
	}

	return true
}

func VerifySvcNotRunning(t *testing.T, svcStatus string) bool {
	pid := extractPID(svcStatus)
	if pid != "0" {
		t.Errorf("service is still running with pid %s", pid)
	}
	return true
}
