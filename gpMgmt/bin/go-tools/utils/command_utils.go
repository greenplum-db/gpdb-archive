package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
)

type CommandBuilder interface {
	BuildExecCommand(gpHome string) *exec.Cmd
}

func NewGpCommand(cmdBuilder CommandBuilder, gpHome string) *exec.Cmd {
	return cmdBuilder.BuildExecCommand(gpHome)
}

func NewGpSourcedCommand(cmdBuilder CommandBuilder, gpHome string) *exec.Cmd {
	cmd := cmdBuilder.BuildExecCommand(gpHome)
	gpSourceFilePath := filepath.Join(gpHome, "greenplum_path.sh")

	return System.ExecCommand("bash", "-c", fmt.Sprintf("source %s && %s", gpSourceFilePath, cmd.String()))
}

func runCommand(cmd *exec.Cmd, filename ...string) (*bytes.Buffer, error) {
	var outfile *os.File
	var err error

	if len(filename) > 0 {
		if len(filename) != 1 {
			return nil, fmt.Errorf("expected only a single filename, got %d", len(filename))
		}

		outfile, err = System.Create(filename[0])
		if err != nil {
			return nil, fmt.Errorf("creating file %q: %w", filename[0], err)
		}
		defer outfile.Close()
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	if outfile != nil {
		cmd.Stdout = io.MultiWriter(stdout, outfile)
		cmd.Stderr = io.MultiWriter(stderr, outfile)
	} else {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}

	gplog.Verbose("Executing command: %s", cmd.String())
	err = cmd.Run()

	if err != nil {
		return stderr, err
	}

	return stdout, err
}

// RunGpCommandAndRedirectOutput executes the command and redirects the stdout and stderr to the given filename
func RunGpCommandAndRedirectOutput(cmdBuilder CommandBuilder, gpHome string, filename string) (*bytes.Buffer, error) {
	return runCommand(NewGpCommand(cmdBuilder, gpHome), filename)
}

// RunGpCommand executes the given command
func RunGpCommand(cmdBuilder CommandBuilder, gpHome string) (*bytes.Buffer, error) {
	out, err := runCommand(NewGpCommand(cmdBuilder, gpHome))
	return out, err
}

// RunGpSourcedCommand sources the greenplum_path.sh before executing the given command
func RunGpSourcedCommand(cmdBuilder CommandBuilder, gpHome string) (*bytes.Buffer, error) {
	out, err := runCommand(NewGpSourcedCommand(cmdBuilder, gpHome))
	return out, err
}

func GetGpUtilityPath(gpHome, utility string) string {
	return path.Join(gpHome, "bin", utility)
}

// GenerateArgs generates command arguments based on the provided CommandBuilder object.
// It inspects the fields of the CommandBuilder struct and appends the corresponding
// flag and value to the args slice if the value is not a default GoLang value.
// The supported data types are string, int, float64, and bool.
// If an unsupported data type is encountered, an error message is logged.
//
// FIXME: GoLang currently uses 0 as the default value for int and float, so for now this function
// will ignore those fields even if you want them to be present in the args list.
//
// Example usage:
// The CommandBuilder struct needs to be defined in the following way,
//
//	type sampleCmd struct {
//		FlagA string  `flag:"--flagA"`
//		FlagB int     `flag:"--flagB"`
//		FlagC float64 `flag:"--flagC"`
//		FlagD bool    `flag:"--flagD"`
//	}
//
// The flag tag is necessary if you want to relate the field value with the
// actual command's flag
//
// To generate the required args, create an object of the CommandBuilder struct and
// pass it to this function:
//
//	cmd := sampleCmd{
//		FlagA: "value",
//		FlagB: 123,
//		FlagD: true,
//	}
//
// This will create the following args based on the struct fields:
// [--flagA value --flagB 123 --flagD]
func GenerateArgs(cmd CommandBuilder) []string {
	var args []string

	// get the value from a pointer field
	value := reflect.ValueOf(cmd)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		tag := value.Type().Field(i).Tag.Get("flag")

		if tag != "" && field.IsValid() {
			switch field.Kind() {
			case reflect.String:
				value := field.Interface().(string)
				if value != "" {
					args = append(args, tag, value)
				}

			case reflect.Int:
				value := field.Interface().(int)
				if value != 0 {
					args = append(args, tag, fmt.Sprintf("%d", value))
				}

			case reflect.Float64:
				value := field.Interface().(float64)
				if value != 0 {
					args = append(args, tag, fmt.Sprintf("%f", value))
				}

			case reflect.Bool:
				if field.Interface().(bool) {
					args = append(args, tag)
				}

			default:
				gplog.Error("unsupported data type %s while generating command arguments for %s",
					field.Kind(), value.Type().Name())
			}
		}
	}

	return args
}
