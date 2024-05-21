package agent_test

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/agent"
	"github.com/greenplum-db/gpdb/gp/idl"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/testutils/exectest"
	"github.com/greenplum-db/gpdb/gp/utils"
)

var (
	errDir = "/tmp/errDir"
)

func TestRemoveDirectory(t *testing.T) {
	_, _, logfile := testhelper.SetupTestLogger()

	agentServer := agent.New(agent.Config{
		GpHome: "gpHome",
	})

	t.Run("does not error out if the directory does not exist", func(t *testing.T) {
		expectedDatadir := "gpseg"
		utils.System.Stat = func(name string) (os.FileInfo, error) {
			if name != expectedDatadir {
				t.Fatalf("got %s, want %s", name, expectedDatadir)
			}

			return nil, os.ErrNotExist
		}
		defer utils.ResetSystemFunctions()

		req := &idl.RemoveDirectoryRequest{
			DataDirectory: expectedDatadir,
		}
		_, err := agentServer.RemoveDirectory(context.Background(), req)

		// Check error
		if err != nil {
			t.Fatalf("stat should ignore file not found error err: %v", err)
		}

	})

	t.Run("Stat succeeds and pgctl status fails but remove directory succeeds", func(t *testing.T) {

		tempdir := t.TempDir()

		req := &idl.RemoveDirectoryRequest{
			DataDirectory: tempdir,
		}

		utils.System.ExecCommand = exectest.NewCommand(exectest.Failure)
		defer utils.ResetSystemFunctions()

		_, err := agentServer.RemoveDirectory(context.Background(), req)
		// Check error
		if err != nil {
			t.Fatalf("unable to remove directory contents err %v", err)
		}
		_, err = os.Stat(tempdir)
		if os.IsNotExist(err) {
			t.Fatalf("unexepected behavior, directory should be retained: %v", err)
		}

	})

	t.Run("Stat succeeds and pgctl status succeeds but pgctl stop fails", func(t *testing.T) {

		tempdir := t.TempDir()
		file, err := os.CreateTemp(tempdir, "*")
		if err != nil {
			t.Fatalf("error creating tempDir err: %v", err)
		}

		req := &idl.RemoveDirectoryRequest{
			DataDirectory: tempdir,
		}

		called := false
		utils.System.ExecCommand = func(name string, arg ...string) *exec.Cmd {
			if !called {
				called = true
				return exectest.NewCommand(exectest.Success)(name, arg...)
			}

			return exectest.NewCommand(exectest.Failure)(name, arg...)
		}
		defer utils.ResetSystemFunctions()
		_, err = agentServer.RemoveDirectory(context.Background(), req)

		testutils.AssertLogMessage(t, logfile, "executing pg_ctl stop")

		if err != nil {
			t.Fatalf("unexepected error err: %v", err)
		}
		// Directory should be present
		_, err = os.Stat(tempdir)
		if os.IsNotExist(err) {
			t.Fatalf("unexepected behavior, directory should be retained : %v", err)
		}

		//File must be removed
		_, err = file.Stat()
		if err != nil {
			t.Fatalf("unexepected behavior, file should be removed : %v", err)
		}

	})

	t.Run("Stat succeeds and pgctl status succeeds but pgctl stop succeeds, remove directory fails", func(t *testing.T) {

		err := testutils.CreateDirectoryWithRemoveFail(errDir)
		if err != nil {
			t.Fatalf("failed to create dummy error directory err: %v", err)
		}

		req := &idl.RemoveDirectoryRequest{
			DataDirectory: errDir,
		}

		utils.System.ExecCommand = exectest.NewCommand(exectest.Success)
		defer utils.ResetSystemFunctions()
		_, err = agentServer.RemoveDirectory(context.Background(), req)

		// Check error
		expectedErrPrefix := "failed to cleanup data directory"
		if err != nil {
			t.Fatalf("got %v, want %v", err, expectedErrPrefix)
		}

	})

	t.Run("Stat succeeds and pgctl status succeeds but pgctl stop succeeds, remove directory succeeds", func(t *testing.T) {

		tempDir := t.TempDir()
		file, err := os.CreateTemp(tempDir, "*")
		if err != nil {
			t.Fatalf("error creating tempDir err: %v", err)
		}
		req := &idl.RemoveDirectoryRequest{
			DataDirectory: tempDir,
		}

		utils.System.ExecCommand = exectest.NewCommand(exectest.Success)
		called := false
		utils.System.ExecCommand = func(name string, arg ...string) *exec.Cmd {
			if !called {
				called = true
				return exectest.NewCommand(exectest.Success)(name, arg...)
			}

			return exectest.NewCommand(exectest.Success)(name, arg...)
		}
		defer utils.ResetSystemFunctions()
		_, err = agentServer.RemoveDirectory(context.Background(), req)
		if err != nil {
			t.Fatalf("unexepected error err: %v", err)
		}
		// Directory should be present
		_, err = os.Stat(tempDir)
		if os.IsNotExist(err) {
			t.Fatalf("unexepected behavior, directory should be retained : %v", err)
		}

		//File must be removed
		_, err = file.Stat()
		if err != nil {
			t.Fatalf("unexepected behavior, file should be removed: %v", err)
		}

	})

}
