package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tetsuo/fortune/internal/wraperr"
	"go.uber.org/zap"
)

// DB wraps a sql.DB. It enhances the original sql.DB by requiring a context argument
// and logging queries with errors.
type DB struct {
	db         *sql.DB
	instanceID string
	logger     *zap.SugaredLogger
}

// Open creates a new DB connection.
func Open(driverName, dsn, instanceID string) (_ *DB, err error) {
	defer wraperr.Wrap(&err, "database.Open(%q, %q)",
		driverName, redactPassword(dsn))

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return New(db, instanceID), nil
}

// New creates a new DB instance.
func New(db *sql.DB, instanceID string) *DB {
	return &DB{db: db, instanceID: instanceID, logger: zap.S()}
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.db.Close()
}

// Exec executes a SQL statement and returns the number of rows affected.
func (db *DB) Exec(ctx context.Context, query string, args ...any) (_ int64, err error) {
	defer logQuery(ctx, db.logger, query, args, db.instanceID)(&err)
	res, err := db.execResult(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("RowsAffected: %v", err)
	}
	return n, nil
}

// execResult executes a SQL statement and returns a sql.Result.
func (db *DB) execResult(ctx context.Context, query string, args ...any) (res sql.Result, err error) {
	return db.db.ExecContext(ctx, query, args...)
}

// QueryRow runs the query and returns a single row.
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	defer logQuery(ctx, db.logger, query, args, db.instanceID)(nil)
	return db.db.QueryRowContext(ctx, query, args...)
}

const OnConflictDoNothing = "ON DUPLICATE KEY UPDATE colA = colA"

// BulkInsert performs a multi-value insert.
func (db *DB) BulkInsert(ctx context.Context, table string, columns []string, values []any, conflictAction string) (err error) {
	defer wraperr.Wrap(&err, "DB.BulkInsert(ctx, %q, %v, [%d values], %q)",
		table, columns, len(values), conflictAction)

	return db.bulkInsert(ctx, table, columns, nil, values, conflictAction, nil)
}

// bulkInsert performs batched inserts.
func (db *DB) bulkInsert(ctx context.Context, table string, columns, returningColumns []string, values []any, conflictAction string, scanFunc func(*sql.Rows) error) (err error) {
	if remainder := len(values) % len(columns); remainder != 0 {
		return fmt.Errorf("modulus of len(values) and len(columns) must be 0: got %d", remainder)
	}

	const maxParameters = 1000
	stride := (maxParameters / len(columns)) * len(columns)
	if stride == 0 {
		return fmt.Errorf("too many columns to insert: %d", len(columns))
	}

	prepare := func(n int) (*sql.Stmt, error) {
		return db.Prepare(ctx, buildInsertQuery(table, columns, returningColumns, n, conflictAction))
	}

	var stmt *sql.Stmt
	for leftBound := 0; leftBound < len(values); leftBound += stride {
		rightBound := leftBound + stride
		if rightBound > len(values) {
			rightBound = len(values)
		}

		stmt, err = prepare(rightBound - leftBound)
		if err != nil {
			return err
		}
		defer stmt.Close()

		valueSlice := values[leftBound:rightBound]
		if returningColumns == nil {
			_, err = stmt.ExecContext(ctx, valueSlice...)
		} else {
			var rows *sql.Rows
			rows, err = stmt.QueryContext(ctx, valueSlice...)
			if err != nil {
				return err
			}
			_, err = processRows(rows, scanFunc)
		}
		if err != nil {
			return fmt.Errorf("running bulk insert query, values[%d:%d]): %w", leftBound, rightBound, err)
		}
	}
	return nil
}

// buildInsertQuery builds a multi-value insert query.
func buildInsertQuery(table string, columns, returningColumns []string, nvalues int, conflictAction string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "INSERT INTO %s (%s) VALUES ", table, strings.Join(columns, ", "))

	// Generate placeholders (?, ?, ?)
	values := make([]string, nvalues/len(columns))
	for i := range values {
		values[i] = "(" + strings.Repeat("?,", len(columns)-1) + "?)"
	}
	b.WriteString(strings.Join(values, ", "))

	if conflictAction != "" {
		b.WriteString(" " + conflictAction)
	}
	if len(returningColumns) > 0 {
		fmt.Fprintf(&b, " RETURNING %s", strings.Join(returningColumns, ", "))
	}
	return b.String()
}

// processRows iterates over query results and executes a callback function on each row.
func processRows(rows *sql.Rows, f func(*sql.Rows) error) (int, error) {
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
		if err := f(rows); err != nil {
			return n, err
		}
	}
	return n, rows.Err()
}

// Prepare prepares a SQL statement for execution.
func (db *DB) Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	defer logQuery(ctx, db.logger, "preparing "+query, nil, db.instanceID)
	return db.db.PrepareContext(ctx, query)
}

var dsnPasswordRegexp = regexp.MustCompile(`(?P<user>\w+):(?P<password>[^@]+)@`)

func redactPassword(dsn string) string {
	return dsnPasswordRegexp.ReplaceAllString(dsn, `${user}:REDACTED@`)
}
