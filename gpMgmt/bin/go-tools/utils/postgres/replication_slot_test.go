package postgres_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpdb/gp/testutils"
	"github.com/greenplum-db/gpdb/gp/utils/postgres"
)

func TestReplicationSlot(t *testing.T) {
	_, _, logfile := testhelper.SetupTestLogger()
	slot := postgres.NewReplicationSlot("sdw1", 1234, "test_slot")

	t.Run("when slot already exists", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		result, err := slot.SlotExists()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Fatalf("expected slot to exist")
		}

		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Checking if replication slot %s exists on %s:%d", slot.Name, slot.Host, slot.Port))
		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Replication slot %s exists on %s:%d", slot.Name, slot.Host, slot.Port))
	})

	t.Run("when slot does not exist", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		result, err := slot.SlotExists()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result {
			t.Fatalf("expected slot to not exist")
		}

		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Checking if replication slot %s exists on %s:%d", slot.Name, slot.Host, slot.Port))
		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Replication slot %s does not exist on %s:%d", slot.Name, slot.Host, slot.Port))
	})

	t.Run("when fails to connect to the database", func(t *testing.T) {
		expectedErr := errors.New("error")
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, _ := testutils.CreateMockDBConnForUtilityMode(t, expectedErr)
			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		_, err := slot.SlotExists()
		expectedErrString := fmt.Sprintf("%v (%s:%d)", expectedErr, slot.Host, slot.Port)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}

		err = slot.DropSlot()
		expectedErrString = fmt.Sprintf("%v (%s:%d)", expectedErr, slot.Host, slot.Port)
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("when a non integer result is returned by the query", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(""))

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		_, err := slot.SlotExists()

		expected := strconv.ErrSyntax
		if !errors.Is(err, expected) {
			t.Fatalf("got %#v, want %#v", err, expected)
		}
	})

	t.Run("when an invalid result is returned by the query", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"col1", "col2"}))

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		_, err := slot.SlotExists()

		expectedErrString := "Too many columns returned from query: got 2 columns, expected 1 column"
		if err.Error() != expectedErrString {
			t.Fatalf("got %v, want %s", err, expectedErrString)
		}
	})

	t.Run("successfully drops the existing slot", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectExec("SELECT pg_drop_replication_slot").WillReturnResult(sqlmock.NewResult(1, 1))

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		err := slot.DropSlot()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Dropping replication slot %s on %s:%d", slot.Name, slot.Host, slot.Port))
		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Successfully dropped replication slot %s on %s:%d", slot.Name, slot.Host, slot.Port))
	})

	t.Run("returns error when not able to execute the drop slot query", func(t *testing.T) {
		expectedErr := errors.New("error")
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.ExpectExec("SELECT pg_drop_replication_slot").WillReturnError(expectedErr)

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		err := slot.DropSlot()
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}

func TestDropSlotIfExists(t *testing.T) {
	_, _, logfile := testhelper.SetupTestLogger()
	slot := postgres.NewReplicationSlot("sdw1", 1234, "test_slot")

	t.Run("successfully drops the slot if it exists", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.MatchExpectationsInOrder(false)
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectExec("SELECT pg_drop_replication_slot").WillReturnResult(sqlmock.NewResult(1, 1))

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		err := postgres.DropSlotIfExists(slot.Host, slot.Port, slot.Name)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Checking if replication slot %s exists on %s:%d", slot.Name, slot.Host, slot.Port))
		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Replication slot %s exists on %s:%d", slot.Name, slot.Host, slot.Port))
		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Dropping replication slot %s on %s:%d", slot.Name, slot.Host, slot.Port))
		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Successfully dropped replication slot %s on %s:%d", slot.Name, slot.Host, slot.Port))
	})

	t.Run("does not drop the slot if it does not exist", func(t *testing.T) {
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.MatchExpectationsInOrder(false)
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectExec("SELECT pg_drop_replication_slot").WillReturnError(errors.New("error")) // this SQL is expected not to be called

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		err := postgres.DropSlotIfExists(slot.Host, slot.Port, slot.Name)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Checking if replication slot %s exists on %s:%d", slot.Name, slot.Host, slot.Port))
		testutils.AssertLogMessage(t, logfile, fmt.Sprintf("Replication slot %s does not exist on %s:%d", slot.Name, slot.Host, slot.Port))
	})

	t.Run("returns appropriate error", func(t *testing.T) {
		expectedErr := errors.New("error")

		// while checking if slot exists
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.MatchExpectationsInOrder(false)
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnError(expectedErr)

			return conn
		})

		err := postgres.DropSlotIfExists(slot.Host, slot.Port, slot.Name)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}

		// while dropping the slot
		postgres.SetNewDBConnFromEnvironment(func(dbname string) *dbconn.DBConn {
			conn, mock := testutils.CreateMockDBConnForUtilityMode(t)
			testhelper.ExpectVersionQuery(mock, "7.0.0")
			mock.MatchExpectationsInOrder(false)
			mock.ExpectQuery("FROM pg_catalog.pg_replication_slots WHERE slot_name").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectExec("SELECT pg_drop_replication_slot").WillReturnError(expectedErr)

			return conn
		})
		defer postgres.ResetNewDBConnFromEnvironment()

		err = postgres.DropSlotIfExists(slot.Host, slot.Port, slot.Name)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got %#v, want %#v", err, expectedErr)
		}
	})
}
