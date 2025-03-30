package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/tetsuo/fortune/internal/wraperr"
)

// recreateDB drops and recreates the database named dbName.
func recreateDB(dbName string) error {
	if err := DropDB(dbName); err != nil {
		return err
	}
	return CreateDB(dbName)
}

// SetupTestDB creates a test database if it does not already exist and migrates it.
func SetupTestDB(dbName string) (_ *DB, err error) {
	defer wraperr.Wrap(&err, "SetupTestDB(%q)", dbName)

	if err := CreateDBIfNotExists(dbName); err != nil {
		return nil, fmt.Errorf("CreateDBIfNotExists(%q): %w", dbName, err)
	}

	// Run existing migration logic without modifying the source
	if isMigrationError, err := TryToMigrate(dbName); err != nil {
		if isMigrationError {
			log.Printf("Migration failed for %s: %v, recreating ", dbName, err)
			if err := recreateDB(dbName); err != nil {
				return nil, fmt.Errorf("recreateDB(%q): %v", dbName, err)
			}
			_, err = TryToMigrate(dbName)
		}
		if err != nil {
			return nil, fmt.Errorf("unfixable error migrating database: %v.\nConsider running ./scripts/drop_test_dbs.sh", err)
		}
	}

	db, err := Open("mysql", DBConnURI(dbName), "test")
	if err != nil {
		return nil, err
	}

	return db, nil
}

// ResetTestDB truncates all data from the given test DB. It should be called after every test that mutates the
func ResetTestDB(db *DB, t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if err := ResetDB(ctx, db); err != nil {
		t.Fatalf("error resetting test DB: %v", err)
	}
}

// RunDBTestsInParallel sets up multiple databases and runs tests in parallel.
func RunDBTestsInParallel(dbBaseName string, numDBs int, m *testing.M, acquirep *func(*testing.T) (*DB, func())) {
	start := time.Now()
	QueryLoggingDisabled = true
	dbs := make(chan *DB, numDBs)
	for i := 0; i < numDBs; i++ {
		db, err := SetupTestDB(fmt.Sprintf("%s_%d", dbBaseName, i))
		if err != nil {
			log.Fatal(err)
		}
		dbs <- db
	}

	*acquirep = func(t *testing.T) (*DB, func()) {
		db := <-dbs
		release := func() {
			ResetTestDB(db, t)
			dbs <- db
		}
		return db, release
	}

	log.Printf("Parallel test setup for %d DBs took %s", numDBs, time.Since(start))
	code := m.Run()
	if len(dbs) != cap(dbs) {
		log.Fatal("not all DBs were released")
	}
	for i := 0; i < numDBs; i++ {
		db := <-dbs
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}
	os.Exit(code)
}
