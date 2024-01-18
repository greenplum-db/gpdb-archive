package testutils

import (
	"fmt"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/dbconn"
	"github.com/jmoiron/sqlx"
)

func ExecQuery(t *testing.T, dbname, query string) *sqlx.Rows {
	t.Helper()

	if dbname == "" {
		dbname = "postgres"
	}
	conn := dbconn.NewDBConnFromEnvironment(dbname)
	err := conn.Connect(1)
	if err != nil {
		t.Fatalf("unexpected error connecting to database: %v", err)
	}
	defer conn.Close()

	result, err := conn.Query(query)
	if err != nil {
		t.Fatalf("unexpected error executing query %q: %v", query, err)
	}

	return result
}

func ExecQueryInUtilityMode(t *testing.T, host string, port int, dbname, query string) *sqlx.Rows {
	t.Helper()

	if dbname == "" {
		dbname = "postgres"
	}
	conn := dbconn.NewDBConnFromEnvironment(dbname)
	conn.Host = host
	conn.Port = port

	err := conn.Connect(1, true)
	if err != nil {
		t.Fatalf("unexpected error connecting to database: %v", err)
	}
	defer conn.Close()

	result, err := conn.Query(query)
	if err != nil {
		t.Fatalf("unexpected error executing query %q: %v", query, err)
	}

	return result
}

func AssertRowCount(t *testing.T, rows *sqlx.Rows, expected int) {
	t.Helper()

	count := 0
	for rows.Next() {
		count += 1
	}

	if count != expected {
		t.Fatalf("unexpected number of rows: got %d, want %d", count, expected)
	}
}

/*
AssertPgConfig asserts for the expected postgres configuration value or GUC
value for a particular segment. Checks for the coordinator segment by default.
*/
func AssertPgConfig(t *testing.T, config string, value string, contentId ...int) {
	content := -1
	if len(contentId) > 1 {
		t.Fatalf("must provide only one content id at a time")
	} else if len(contentId) == 1 {
		content = contentId[0]
	}

	conn := dbconn.NewDBConnFromEnvironment("postgres")
	err := conn.Connect(1)
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}
	defer conn.Close()

	var result []string
	err = conn.Select(&result, fmt.Sprintf("SELECT paramvalue FROM gp_toolkit.gp_param_setting('%s') WHERE paramsegment = %d", config, content))
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	if len(result) != 1 {
		t.Fatalf("unexpected number of rows, want only 1")
	}

	if result[0] != value {
		t.Fatalf("pg config %q: got %q, want %q", config, result[0], value)
	}
}
