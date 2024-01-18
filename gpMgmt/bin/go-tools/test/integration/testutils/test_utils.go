package testutils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/hub"
	"github.com/greenplum-db/gpdb/gp/utils"
)

type Command struct {
	host   string
	cmdStr string
	args   []string
}

type CmdResult struct {
	OutputMsg string
	ExitCode  int
}

const ExitCode1 = 1

func ConfigureAndStartServices(hostfile string) error {
	result, err := RunConfigure(true, "--hostfile", hostfile)
	if err != nil {
		return fmt.Errorf("failed to configure the services: %v, %v", result.OutputMsg, err)
	}

	result, err = RunStart("services")
	if err != nil {
		return fmt.Errorf("failed to start the services: %v, %v", result.OutputMsg, err)
	}

	startTime := time.Now()
	timeout := 10 * time.Second

	for {
		result, err := RunStatus("services")
		if err == nil {
			return nil
		}

		if time.Since(startTime) >= timeout {
			return fmt.Errorf("failed to configure services after %v seconds: %v, %v", timeout, result.OutputMsg, err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func RunConfigure(useCert bool, params ...string) (CmdResult, error) {
	var args []string

	if useCert {
		args = append([]string{"configure"}, CertificateParams...)
		args = append(args, params...)
	} else {
		args = append([]string{"configure"}, params...)
	}

	genCmd := Command{
		cmdStr: constants.DefaultServiceName,
		args:   args,
	}
	return runCmd(genCmd)
}

func RunStart(params ...string) (CmdResult, error) {
	params = append([]string{"start"}, params...)
	genCmd := Command{
		cmdStr: constants.DefaultServiceName,
		args:   params,
	}
	return runCmd(genCmd)
}

func RunStop(params ...string) (CmdResult, error) {
	params = append([]string{"stop"}, params...)
	genCmd := Command{
		cmdStr: constants.DefaultServiceName,
		args:   params,
	}
	return runCmd(genCmd)
}

func RunStatus(params ...string) (CmdResult, error) {
	params = append([]string{"status"}, params...)
	genCmd := Command{
		cmdStr: constants.DefaultServiceName,
		args:   params,
	}
	return runCmd(genCmd)
}

func RunInitCluster(params ...string) (CmdResult, error) {
	params = append([]string{"init", "cluster"}, params...)
	genCmd := Command{
		cmdStr: constants.DefaultServiceName,
		args:   params,
	}
	return runCmd(genCmd)
}

func RunGpStatus(params ...string) (CmdResult, error) {
	allParams := append([]string{"gpstate"}, params...)

	genCmd := Command{
		cmdStr: allParams[0],  
		args:   allParams[1:],
	}

	return runCmd(genCmd)
}

func RunGpCheckCat(params ...string) (CmdResult, error) {
	allParams := append([]string{"gpcheckcat"}, params...)

	genCmd := Command{
		cmdStr: allParams[0],
		args:   allParams[1:],
	}

	return runCmd(genCmd)
}

func RunGpStop(params ...string) (CmdResult, error) {
	allParams := append([]string{"gpstop", "-a"}, params...)

	genCmd := Command{
		cmdStr: allParams[0],
		args:   allParams[1:],
	}

	return runCmd(genCmd)
}

func RunGpStart(params ...string) (CmdResult, error) {
	allParams := append([]string{"gpstart", "-a"}, params...)

	genCmd := Command{
		cmdStr: allParams[0],
		args:   allParams[1:],
	}

	return runCmd(genCmd)
}

func DeleteCluster() (CmdResult, error) {
	genCmd := Command{
		cmdStr: "bash",
		args:   []string{"-c", "echo -e 'y\ny' | gpdeletesystem"},
	}
	return runCmd(genCmd)
}

func ParseConfig(configFile string) hub.Config {
	gpConfig := hub.Config{}
	gpConfig.Credentials = &utils.GpCredentials{}
	config, _ := os.Open(configFile)
	defer config.Close()
	byteValue, _ := io.ReadAll(config)

	_ = json.Unmarshal(byteValue, &gpConfig)

	return gpConfig
}

func CleanupFilesOnHub(files ...string) {
	for _, f := range files {
		_ = os.RemoveAll(f)
	}
}

func CleanupFilesOnAgents(file string, hosts []string) {
	cmdStr := fmt.Sprintf("/bin/bash -c 'rm -rf %s && echo $?'", file)
	for _, host := range hosts {
		cmd := exec.Command("ssh", host, cmdStr)
		_, _ = cmd.CombinedOutput()
	}
}

func CpCfgWithoutCertificates(name string) error {
	cfg := ParseConfig(DefaultConfigurationFile)
	cfg.Credentials = &utils.GpCredentials{}
	content, _ := json.Marshal(cfg)
	return os.WriteFile(name, content, 0777)
}

func extractPID(outMessage string) string {
	pidRegex1 := regexp.MustCompile(`"PID"\s*=\s*(\d+);`)
	pidRegex2 := regexp.MustCompile(`MainPID=(\d+)`)
	pidRegex3 := regexp.MustCompile(`\b(\d+)\s+\w+\b`)
	if match := pidRegex1.FindStringSubmatch(outMessage); len(match) >= 2 {
		return match[1]
	} else if match = pidRegex2.FindStringSubmatch(outMessage); len(match) >= 2 {
		return match[1]
	}

	for _, line := range strings.Split(outMessage, "\n") {
		if strings.Contains(line, "(LISTEN)") && pidRegex3.MatchString(line) {
			match := pidRegex3.FindStringSubmatch(line)
			if len(match) > 1 {
				return match[1]
			}
		}
	}

	return "0"
}

func runCmd(cmd Command) (CmdResult, error) {
	var cmdObj *exec.Cmd
	if cmd.host == "" || cmd.host == DefaultHost {
		cmdObj = exec.Command(cmd.cmdStr, cmd.args...)
	} else {
		subCmd := exec.Command(cmd.cmdStr, cmd.args...)
		cmdObj = exec.Command("ssh", cmd.host, subCmd.String())
	}

	out, err := cmdObj.CombinedOutput()
	result := CmdResult{
		OutputMsg: string(out),
		ExitCode:  cmdObj.ProcessState.ExitCode(),
	}

	return result, err
}

func GetServiceDetails(p utils.Platform) (string, string, string) {
	serviceDir := fmt.Sprintf(p.GetDefaultServiceDir(), os.Getenv("USER"))
	serviceExt := p.(utils.GpPlatform).ServiceExt
	serviceCmd := p.(utils.GpPlatform).ServiceCmd

	return serviceDir, serviceExt, serviceCmd
}

func UnloadSvcFile(cmd string, file string) {
	genCmd := Command{
		cmdStr: cmd,
	}
	if cmd == "launchctl" {
		genCmd.args = []string{"unload", file}

	} else {
		genCmd.args = []string{"--user", "stop", file}
	}
	_, _ = runCmd(genCmd)
}

func DisableandDeleteServiceFiles(p utils.Platform) {
	serviceDir, serviceExt, serviceCmd := GetServiceDetails(p)
	filesToUnload := GetSvcFiles(serviceDir, serviceExt)
	for _, filePath := range filesToUnload {
		UnloadSvcFile(serviceCmd, filepath.Base(filePath))
		_ = os.RemoveAll(filePath)
	}
}

func GetSvcFiles(svcDir string, svcExtention string) []string {
	pattern := filepath.Join(svcDir, fmt.Sprintf("*.%s", svcExtention))
	fileList, _ := filepath.Glob(pattern)
	return fileList
}

func InitService(hostfile string, params []string) {
	_, _ = RunConfigure(false, append(
		[]string{
			"--hostfile", hostfile,
		},
		params...)...)
	time.Sleep(5 * time.Second)
}

func CopyFile(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dest, input, 0644)
	if err != nil {
		return err
	}
	return nil
}

func GetSvcStatusOnHost(p utils.GpPlatform, serviceName string, host string) (CmdResult, error) {
	args := []string{p.UserArg, p.StatusArg, serviceName}

	if p.OS == "darwin" {
		args = args[1:]
	}
	genCmd := Command{
		cmdStr: p.ServiceCmd,
		args:   args,
	}
	genCmd.host = host

	return runCmd(genCmd)
}

func GetListeningProcess(port int, host string) string {
	var output string
	if host == DefaultHost || host == "" {
		cmd := exec.Command("lsof", fmt.Sprintf("-i:%d", port))
		out, _ := cmd.CombinedOutput()
		output = string(out)
	} else {
		genCmd := Command{
			cmdStr: fmt.Sprintf("lsof -i:%d", port),
			host:   host,
		}
		result, _ := runCmd(genCmd)
		output = result.OutputMsg
	}

	return extractPID(output)
}

func ExtractStatusData(data string) map[string]map[string]string {
	lines := strings.Split(data, "\n")
	dataMap := make(map[string]map[string]string)

	for i := 1; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) >= 4 {
			role := fields[0] // agent or hub
			host := fields[1]
			pid := fields[3]

			// Create a map for the role if it doesn't exist
			if _, ok := dataMap[role]; !ok {
				dataMap[role] = make(map[string]string)
			}

			// Add host and pid to the role's map
			dataMap[role][host] = pid
		}
	}

	return dataMap
}

func StructToString(s interface{}) string {
	return structToString(reflect.ValueOf(s), 0)
}

func structToString(value reflect.Value, indentLevel int) string {
	if value.Kind() != reflect.Struct {
		return ""
	}

	var result string
	result += fmt.Sprintf("%s {", value.Type().Name())

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldName := value.Type().Field(i).Name

		if field.Kind() == reflect.Struct {
			// Recursively handle nested structs
			nestedString := structToString(field, indentLevel+1)
			result += fmt.Sprintf("\n%s%s: %s", strings.Repeat("    ", indentLevel+1), fieldName, nestedString)
		} else {
			result += fmt.Sprintf("%s: %v", fieldName, field.Interface())
		}

		if i < value.NumField()-1 {
			result += ", "
		}
	}

	result += "}"
	return result
}

func GetHostListFromFile(hostfile string) []string {
	content, _ := os.ReadFile(hostfile)

	return strings.Fields(string(content))
}

func GetTempFile(t *testing.T, name string) string {
	dir := t.TempDir()
	return filepath.Join(dir, name)
}
