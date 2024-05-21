package utils_test

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"

	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/utils"
)

var (
	dummyDir    = "/tmp/xyz"
	nonExistent = "/path/to/nonexistent/directory"
)

func TestCheckIfPortFree(t *testing.T) {
	testhelper.SetupTestLogger()
	t.Run("returns no error when port not in use", func(t *testing.T) {
		_, err := utils.CheckIfPortFree("localhost", "9946")
		if err != nil {
			t.Fatalf("Got error %v, expected no error", err)
		}
	})
	t.Run("returns error when port not in use", func(t *testing.T) {
		portNum := "9946"
		hostName := "localhost"
		strErr := "bind: address already in use"
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", hostName, portNum))
		if err != nil {
			t.Fatalf("Failed to create test listener on port %s:%s", hostName, portNum)
		}
		defer listener.Close()

		_, err = utils.CheckIfPortFree(hostName, portNum)
		if err == nil || !strings.Contains(err.Error(), strErr) {
			t.Fatalf("Got error %v, expected:%sr", err, strErr)
		}
	})
}

func TestGetListDifference(t *testing.T) {
	testhelper.SetupTestLogger()
	t.Run("returns correct diff when both lists are same", func(t *testing.T) {
		list1 := []string{"s1", "s2", "s3"}
		list2 := []string{"s1", "s2", "s3"}
		var expResult []string
		result := utils.GetListDifference(list1, list2)
		if !reflect.DeepEqual(result, expResult) {
			t.Fatalf("got:%v, expected:%v", result, list1)
		}
	})
	t.Run("returns correct diff when list1 and list2 has some common elements", func(t *testing.T) {
		list1 := []string{"s1", "s2", "s3"}
		list2 := []string{"s1", "s2"}
		expResult := []string{"s3"}
		result := utils.GetListDifference(list1, list2)
		if !reflect.DeepEqual(result, expResult) {
			t.Fatalf("got:%v, expected:%v", result, list1)
		}
	})
	t.Run("returns correct diff when list2 has some extra elements", func(t *testing.T) {
		list1 := []string{"s1", "s2", "s3"}
		list2 := []string{"s2", "s1", "s4", "s5"}
		expResult := []string{"s3"}
		result := utils.GetListDifference(list1, list2)
		if !reflect.DeepEqual(result, expResult) {
			t.Fatalf("got:%v, expected:%v", result, list1)
		}
	})
	t.Run("returns correct diff when list2 has no elements", func(t *testing.T) {
		list1 := []string{"s1", "s2", "s3"}
		list2 := []string{}
		expResult := list1
		result := utils.GetListDifference(list1, list2)
		if !reflect.DeepEqual(result, expResult) {
			t.Fatalf("got:%v, expected:%v", result, list1)
		}
	})
	t.Run("returns correct diff when list2 has some extra elements", func(t *testing.T) {
		list1 := []string{}
		list2 := []string{"s1", "s2", "s4", "s5"}
		expResult := list1
		result := utils.GetListDifference(list1, list2)
		if len(expResult) != 0 {
			t.Fatalf("got:%v, expected:%v", result, list1)
		}
	})
}

func TestWriteLinesToFile(t *testing.T) {
	testhelper.SetupTestLogger()

	t.Run("succesfully writes to the file", func(t *testing.T) {
		filename := filepath.Join(os.TempDir(), "test")
		defer os.Remove(filename)

		lines := []string{"line1", "line2", "line3"}
		err := utils.WriteLinesToFile(filename, lines)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		expected := strings.Join(lines, "\n")
		testutils.AssertFileContents(t, filename, expected)
	})

	t.Run("errors out when not able to create the file", func(t *testing.T) {
		file, err := os.CreateTemp("", "")
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
		defer os.Remove(file.Name())

		err = os.Chmod(file.Name(), 0000)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		err = utils.WriteLinesToFile(file.Name(), []string{})

		expectedErr := os.ErrPermission
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})

	t.Run("errors out when not able to write to the file", func(t *testing.T) {
		utils.System.Create = func(name string) (*os.File, error) {
			_, writer, _ := os.Pipe()
			writer.Close()

			return writer, nil
		}
		defer utils.ResetSystemFunctions()

		err := utils.WriteLinesToFile("", []string{})

		expectedErr := os.ErrClosed
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}

func TestAppendLinesToFile(t *testing.T) {
	testhelper.SetupTestLogger()

	t.Run("succesfully appends to the file", func(t *testing.T) {
		file, err := os.CreateTemp("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(file.Name())

		existingLines := []string{"line1", "line2", "line3"}
		_, err = file.WriteString(strings.Join(existingLines, "\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		newLines := []string{"line4", "line5", "line6"}
		err = utils.AppendLinesToFile(file.Name(), newLines)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		expected := strings.Join(append(existingLines, newLines...), "\n")
		testutils.AssertFileContents(t, file.Name(), expected)
	})

	t.Run("errors out when the file does not exist", func(t *testing.T) {
		err := utils.AppendLinesToFile("non_exisitng_file", []string{})

		expectedErr := os.ErrNotExist
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}
func TestCreateAppendLinesToFile(t *testing.T) {

	t.Run("Successfully creates file if it does not exist", func(t *testing.T) {
		filename := "/tmp/xyz.txt"

		newLines := []string{"line4", "line5", "line6"}

		err := utils.CreateAppendLinesToFile(filename, newLines)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		err = utils.System.RemoveAll(filename)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

	})
	t.Run("succesfully appends to the file", func(t *testing.T) {

		file, err := os.CreateTemp("", "")
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}
		defer os.Remove(file.Name())

		existingLines := []string{"line1", "line2", "line3"}
		_, err = file.WriteString(strings.Join(existingLines, "\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = file.WriteString("\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		newLines := []string{"line4", "line5", "line6"}
		err = utils.CreateAppendLinesToFile(file.Name(), newLines)
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		expected := strings.Join(append(existingLines, newLines...), "\n")
		testutils.AssertFileContents(t, file.Name(), expected)
	})

	t.Run("errors out when not able to write to the file", func(t *testing.T) {

		utils.System.OpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
			_, writer, _ := os.Pipe()
			writer.Close()

			return writer, nil
		}
		defer utils.ResetSystemFunctions()

		err := utils.CreateAppendLinesToFile("", []string{})

		expectedErrPrefix := "open : no such file or directory"
		if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
			t.Fatalf("got %s, want %s", err.Error(), expectedErrPrefix)
		}
	})
}

func TestGetHostAddrsNoLoopback(t *testing.T) {
	testhelper.SetupTestLogger()
	t.Run("returns the correct address without loopback", func(t *testing.T) {
		utils.System.InterfaceAddrs = func() ([]net.Addr, error) {
			_, addr1, _ := net.ParseCIDR("192.0.1.0/24")
			_, addr2, _ := net.ParseCIDR("2001:db8::/32")
			_, loopbackAddrIp4, _ := net.ParseCIDR("127.0.0.1/8")
			_, loopbackAddrIp6, _ := net.ParseCIDR("::1/128")

			return []net.Addr{addr1, addr2, loopbackAddrIp4, loopbackAddrIp6}, nil
		}
		defer utils.ResetSystemFunctions()

		result, err := utils.GetHostAddrsNoLoopback()
		if err != nil {
			t.Fatalf("unexpected error: %#v", err)
		}

		expected := []string{"192.0.1.0/24", "2001:db8::/32"}
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("got %v, want %v", result, expected)
		}
	})

	t.Run("errors out when not able to get the host address", func(t *testing.T) {
		expectedErr := errors.New("error")
		utils.System.InterfaceAddrs = func() ([]net.Addr, error) {
			return nil, expectedErr
		}
		defer utils.ResetSystemFunctions()

		_, err := utils.GetHostAddrsNoLoopback()
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}

func TestReadEntriesFromFile(t *testing.T) {
	// Create a temporary test file
	tempFile, err := os.CreateTemp("", "testfile.txt")
	if err != nil {
		t.Fatalf("error creating temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write some content to the temporary file
	content := "line 1\nline 2\nline 3"
	_, err = tempFile.WriteString(content)
	if err != nil {
		t.Fatalf("error writing to temporary file: %v", err)
	}
	tempFile.Close()

	// Test case 1: file exists and contains entries
	t.Run("file exists and contains entries", func(t *testing.T) {
		entries, err := utils.ReadEntriesFromFile(tempFile.Name())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		expectedEntries := []string{"line 1", "line 2", "line 3"}
		if !reflect.DeepEqual(entries, expectedEntries) {
			t.Fatalf("expected entries %v, got %v", expectedEntries, entries)
		}

	})

	// Test case 2: file exists but is empty
	t.Run("file exists but is empty", func(t *testing.T) {
		emptyFile, err := os.CreateTemp("", "emptyfile.txt")
		if err != nil {
			t.Fatalf("error creating temporary file: %v", err)
		}
		defer os.Remove(emptyFile.Name())
		emptyFile.Close()

		emptyEntries, err := utils.ReadEntriesFromFile(emptyFile.Name())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(emptyEntries) != 0 {
			t.Fatalf("expected empty entries, got %v", emptyEntries)
		}
	})

	// Test case 3: file doesn't exist
	t.Run("file doesnt exist", func(t *testing.T) {
		_, err = utils.ReadEntriesFromFile("nonexistentfile.txt")
		if err == nil {
			t.Fatalf("expected an error, got none")
		}

		// Test case 4: error while reading file
		_, err = utils.ReadEntriesFromFile("/root/forbidden.txt")
		if err == nil {
			t.Fatalf("expected an error, got none")
		}
	})
}

func TestRemoveDirContents(t *testing.T) {

	// Test case 1: Directory does not exist
	t.Run("Directory does not exist", func(t *testing.T) {

		expectedErr := "open /path/to/nonexistent/directory: no such file or directory"
		err := utils.RemoveDirContents(nonExistent)
		if err != nil {
			if !strings.HasPrefix(err.Error(), expectedErr) {
				t.Fatalf("got %s, want %s", err.Error(), expectedErr)
			}
		}

	})

	//Test case 2: Directory is empty
	t.Run("Empty directory", func(t *testing.T) {

		tempDir := t.TempDir()

		err := utils.RemoveDirContents(tempDir)
		if err != nil {
			t.Fatalf("function should succeed if directory is empty: %v", err)
		}
	})

	//Test case 3: Successfully removes all files in directory
	t.Run("Successfully removed files", func(t *testing.T) {

		//Create file if it doesnt exist

		if err := os.MkdirAll(dummyDir, 0777); err != nil {
			err = os.Chmod(dummyDir, 0777)
			t.Fatalf("unable to create dummy directory: %v", err)
		}
		defer os.RemoveAll(dummyDir)

		//Create dummy files
		_, err := os.Create(filepath.Join(dummyDir, "file1"))
		if err != nil {
			t.Fatalf("unable to create files in dummy directory: %v", err)
		}

		_, err = os.Create(filepath.Join(dummyDir, "file2"))
		if err != nil {
			t.Fatalf("unable to create files in dummy directory: %v", err)
		}

		err = utils.RemoveDirContents(dummyDir)
		if err != nil {
			t.Fatalf("unable to remove contents of dummpDir err: %v", err)
		}

	})

	//Test case 4: Successfully removes file but not directory
	t.Run("Successfully removed files but not directory", func(t *testing.T) {

		//Create file if it doesnt exist

		if err := os.MkdirAll(dummyDir, 0777); err != nil {
			err = os.Chmod(dummyDir, 0777)
			t.Fatalf("unable to create dummy directory: %v", err)
		}
		defer os.RemoveAll(dummyDir)

		//Create dummy files
		_, err := os.Create(filepath.Join(dummyDir, "file1"))
		if err != nil {
			t.Fatalf("unable to create files in dummy directory: %v", err)
		}

		_, err = os.Create(filepath.Join(dummyDir, "file2"))
		if err != nil {
			t.Fatalf("unable to create files in dummy directory: %v", err)
		}

		err = utils.RemoveDirContents(dummyDir)
		if err != nil {
			t.Fatalf("unable to remove contents of dummpDir err: %v", err)
		}

		_, err = utils.System.Stat(dummyDir)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

	})

}
