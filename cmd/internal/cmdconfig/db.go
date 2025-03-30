package cmdconfig

import (
	"context"
	"fmt"

	"contrib.go.opencensus.io/integrations/ocsql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tetsuo/fortune/internal/database"
	"go.uber.org/zap"
)

// OpenDB opens the MySQL database specified by the config.
func OpenDB(ctx context.Context, instanceID string, cfg database.DBConfig) (_ *database.DB, err error) {
	log := zap.S()

	// Wrap the MySQL driver with OpenCensus instrumentation.
	ocDriver, err := database.RegisterOCWrapper("mysql", ocsql.WithAllTraceOptions())
	if err != nil {
		return nil, fmt.Errorf("database.RegisterOCWrapper: %v", err)
	}

	log.With(
		"host", cfg.DBHost,
		"port", cfg.DBPort,
		"name", cfg.DBName,
		"user", cfg.DBUser,
	).Infof("opening database on host %s", cfg.DBHost)

	db, err := database.Open(ocDriver, cfg.DSN(), instanceID)
	if err != nil {
		return nil, err
	}

	log.Debug("database open finished")

	return db, nil
}
