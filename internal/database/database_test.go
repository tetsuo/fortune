package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

const testTimeout = 5 * time.Second
const testDBName = "fortune_mysql_test"

var testDB *DB

func TestMain(m *testing.M) {
	if err := CreateDBIfNotExists(testDBName); err != nil {
		log.Fatal(err)
	}

	var err error
	log.Printf("with driver %q", "mysql")
	testDB, err = Open("mysql", DBConnURI(testDBName), "test")
	if err != nil {
		log.Fatalf("Open: %v %[1]T", err)
	}
	code := m.Run()
	if err := testDB.Close(); err != nil {
		log.Fatal(err)
	}
	os.Exit(code)
}

func TestBulkInsert(t *testing.T) {
	table := "test_bulk_insert"

	for _, test := range []struct {
		name           string
		columns        []string
		values         []any
		conflictAction string
		wantErr        bool
		wantCount      int
	}{
		{

			name:      "test-one-row",
			columns:   []string{"colA"},
			values:    []any{"valueA"},
			wantCount: 1,
		},
		{

			name:      "test-multiple-rows",
			columns:   []string{"colA"},
			values:    []any{"valueA1", "valueA2", "valueA3"},
			wantCount: 3,
		},
		{

			name:    "test-invalid-column-name",
			columns: []string{"invalid_col"},
			values:  []any{"valueA"},
			wantErr: true,
		},
		{

			name:    "test-mismatch-num-cols-and-vals",
			columns: []string{"colA", "colB"},
			values:  []any{"valueA1", "valueB1", "valueA2"},
			wantErr: true,
		},
		{

			name:    "test-conflict",
			columns: []string{"colA"},
			values:  []any{"valueA", "valueA"},
			wantErr: true,
		},
		{

			name:           "test-conflict-do-nothing",
			columns:        []string{"colA"},
			values:         []any{"valueA", "valueA"},
			conflictAction: OnConflictDoNothing,
			wantCount:      1,
		},
		{
			// This should execute the statement
			// INSERT INTO series (path) VALUES ('''); TRUNCATE series CASCADE;)');
			// which will insert a row with path value:
			// '); TRUNCATE series CASCADE;)
			// Rather than the statement
			// INSERT INTO series (path) VALUES (''); TRUNCATE series CASCADE;));
			// which would truncate most tables in the database.
			name:           "test-sql-injection",
			columns:        []string{"colA"},
			values:         []any{fmt.Sprintf("''); TRUNCATE %s CASCADE;))", table)},
			conflictAction: OnConflictDoNothing,
			wantCount:      1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			createQuery := fmt.Sprintf(`CREATE TABLE %s (
					colA VARCHAR(255) NOT NULL, -- Changed from TEXT to VARCHAR(255)
					colB TEXT,
					PRIMARY KEY (colA)
			);`, table)

			if _, err := testDB.Exec(ctx, createQuery); err != nil {
				t.Fatal(err)
			}
			defer func() {
				dropTableQuery := fmt.Sprintf("DROP TABLE %s;", table)
				if _, err := testDB.Exec(ctx, dropTableQuery); err != nil {
					t.Fatal(err)
				}
			}()

			err := testDB.BulkInsert(ctx, table, test.columns, test.values, test.conflictAction)

			if test.wantErr && err == nil || !test.wantErr && err != nil {
				t.Errorf("got error %v, wantErr %t", err, test.wantErr)
			}
			if err != nil {
				return
			}
			if test.wantCount != 0 {
				var count int
				query := "SELECT COUNT(*) FROM " + table
				row := testDB.QueryRow(ctx, query)
				err := row.Scan(&count)
				if err != nil {
					t.Fatalf("testDB.queryRow(%q): %v", query, err)
				}
				if count != test.wantCount {
					t.Errorf("testDB.queryRow(%q) = %d; want = %d", query, count, test.wantCount)
				}
			}
		})

	}
}
