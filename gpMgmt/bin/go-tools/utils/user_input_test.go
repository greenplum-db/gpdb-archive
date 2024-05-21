package utils_test

import (
	"os"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/constants"
	"github.com/greenplum-db/gpdb/gp/utils"
)

func TestAskUserYesNo(t *testing.T) {
	testhelper.SetupTestLogger()

	// Mock user input for testing
	mockInputYes := "yes\n"
	mockInputNo := "no\n"
	mockInputInvalid := "invalid\n"

	/// Set up mock stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	t.Run("User inputs yes", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, err := w.WriteString(mockInputYes)
		if err != nil {
			t.Errorf("WriteString error %v", err)
		}
		if expected := utils.AskUserYesNo(constants.UserInputWaitDurtion); !expected {
			t.Errorf("AskUserYesNo() = %v, actual %v", expected, true)
		}
		w.Close()
	})

	t.Run("User Inputs No", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, err := w.WriteString(mockInputNo)
		if err != nil {
			t.Errorf("WriteString error %v", err)
		}
		if expected := utils.AskUserYesNo(constants.UserInputWaitDurtion); expected {
			t.Errorf("AskUserYesNo() = %v, actual %v", expected, false)
		}
		w.Close()
	})

	t.Run("User Inputs Invalid", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, err := w.WriteString(mockInputInvalid)
		if err != nil {
			t.Errorf("WriteString error %v", err)
		}
		// Simulate an invalid input followed by a valid one
		if expected := utils.AskUserYesNo(constants.UserInputWaitDurtion); expected {
			t.Errorf("AskUserYesNo() = %v, actual %v", expected, false)
		}
		_, err = w.WriteString(mockInputYes)
		if err != nil {
			t.Errorf("WriteString error %v", err)
		}
		//User inputs valid one
		if expected := utils.AskUserYesNo(constants.UserInputWaitDurtion); !expected {
			t.Errorf("AskUserYesNo() = %v, actual %v", !expected, true)
		}
		w.Close()
	})
}
