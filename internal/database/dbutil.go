package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"

	// imported to register the mysql migration driver
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"

	// imported to register the file source migration driver
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// imported to register the MySQL database driver
	_ "github.com/go-sql-driver/mysql"
)

// DBConnURI generates a MySQL connection string in URI format.
func DBConnURI(dbName string) string {
	var (
		user     = "root"
		password = "example"
		host     = "localhost"
		port     = "3306"
	)
	cs := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		url.QueryEscape(user), url.QueryEscape(password), host, port, dbName)
	return cs
}

// MultiErr can be used to combine one or more errors into a single error.
type MultiErr []error

func (m MultiErr) Error() string {
	var sb strings.Builder
	for _, err := range m {
		sep := ""
		if sb.Len() > 0 {
			sep = "|"
		}
		if err != nil {
			sb.WriteString(sep + err.Error())
		}
	}
	return sb.String()
}

// ConnectAndExecute connects to the MySQL database specified by uri and executes dbFunc.
func ConnectAndExecute(uri string, dbFunc func(*sql.DB) error) (outerErr error) {
	db, err := sql.Open("mysql", uri)
	if err == nil {
		err = db.Ping()
	}
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			outerErr = MultiErr{outerErr, err}
		}
	}()
	return dbFunc(db)
}

// CreateDB creates a new MySQL database.
func CreateDB(dbName string) error {
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", dbName)
	return ConnectAndExecute(DBConnURI(""), func(db *sql.DB) error {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("error creating %q: %v", dbName, err)
		}
		return nil
	})
}

// DropDB drops the MySQL database named dbName.
func DropDB(dbName string) error {
	query := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName)
	return ConnectAndExecute(DBConnURI(""), func(db *sql.DB) error {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("error dropping %q: %v", dbName, err)
		}
		return nil
	})
}

// CreateDBIfNotExists checks whether the given dbName is an existing database,
// and creates one if not.
func CreateDBIfNotExists(dbName string) error {
	exists, err := checkIfDBExists(dbName)
	if err != nil || exists {
		return err
	}

	log.Printf("database %q does not exist, creating.", dbName)
	return CreateDB(dbName)
}

// checkIfDBExists checks if a MySQL database exists.
func checkIfDBExists(dbName string) (bool, error) {
	var exists bool
	err := ConnectAndExecute(DBConnURI(""), func(db *sql.DB) error {
		rows, err := db.Query("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", dbName)
		if err != nil {
			return err
		}
		defer rows.Close()
		if rows.Next() {
			exists = true
			return nil
		}
		return rows.Err()
	})
	return exists, err
}

// TryToMigrate attempts to migrate the database named dbName to the latest
// migration. If this operation fails in the migration step, it returns
// isMigrationError=true to signal that the database should be recreated.
func TryToMigrate(dbName string) (isMigrationError bool, outerErr error) {
	dbURI := fmt.Sprintf("mysql://%s", DBConnURI(dbName))
	source := migrationsSource()
	m, err := migrate.New(source, dbURI)
	if err != nil {
		return false, fmt.Errorf("migrate.New(): %v", err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			outerErr = MultiErr{outerErr, srcErr, dbErr}
		}
	}()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return true, fmt.Errorf("m.Up() %q: %v", source, err)
	}
	return false, nil
}

func migrationsSource() string {
	migrationsDir := testDataPath("../../etc/migrations")
	return "file://" + filepath.ToSlash(migrationsDir)
}

// ResetDB truncates all data from the given MySQL test DB.
func ResetDB(ctx context.Context, db *DB) error {
	if _, err := db.Exec(ctx, `SET FOREIGN_KEY_CHECKS = 0;`); err != nil {
		return fmt.Errorf("error resetting test DB: %v", err)
	}
	if _, err := db.Exec(ctx, `TRUNCATE TABLE fortune_cookies;`); err != nil {
		return fmt.Errorf("error resetting test DB: %v", err)
	}
	if _, err := db.Exec(ctx, `SET FOREIGN_KEY_CHECKS = 1;`); err != nil {
		return fmt.Errorf("error resetting test DB: %v", err)
	}
	return nil
}

// TestDataPath returns a path corresponding to a path relative to the calling
// test file. For convenience, rel is assumed to be "/"-delimited.
//
// It panics on failure.
func testDataPath(rel string) string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("unable to determine relative path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), filepath.FromSlash(rel)))
}
