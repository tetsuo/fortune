package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/caarlos0/env"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/tetsuo/fortune/internal/database"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: db [cmd]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  create: creates a new database. It does not run migrations\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  migrate: runs all migrations \n")
		fmt.Fprintf(flag.CommandLine.Output(), "  drop: drops database\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  truncate: truncates all tables in database\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  recreate: drop, create and run migrations\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Database name is set using $DATABASE_NAME.\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	cfg := database.DBConfig{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?multiStatements=true&parseTime=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	if err := run(context.Background(), flag.Args()[0], cfg.DBName, dsn); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cmd, dbName, connectionInfo string) error {
	switch cmd {
	case "create":
		return create(dbName)
	case "migrate":
		return migrateUp(connectionInfo)
	case "drop":
		return drop(dbName)
	case "recreate":
		return recreate(dbName)
	case "truncate":
		return truncate(ctx, connectionInfo)
	default:
		return fmt.Errorf("unsupported arg: %q", cmd)
	}
}

func create(dbName string) error {
	if err := database.CreateDBIfNotExists(dbName); err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "database exists") {
			log.Print(strings.TrimPrefix(err.Error(), "error creating "))
			return nil
		}
		return err
	}
	log.Printf("Database created: %q", dbName)
	return nil
}

func migrateUp(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("failed to create MySQL migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://etc/migrations",
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migration
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("Database migration successful!")
	return nil
}

func drop(dbName string) error {
	err := database.DropDB(dbName)
	if err != nil {
		if strings.Contains(err.Error(), "Unknown database") {
			log.Printf("Database does not exist: %q", dbName)
			return nil
		}
		return err
	}
	log.Printf("Dropped database: %q", dbName)
	return nil
}

func recreate(dbName string) error {
	if err := drop(dbName); err != nil {
		return err
	}
	if err := database.CreateDB(dbName); err != nil {
		return err
	}
	return migrateUp(fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?multiStatements=true&parseTime=true",
		"root", "example", "127.0.0.1", "3306", dbName))
}

func truncate(ctx context.Context, connectionInfo string) error {
	db, err := database.Open("mysql", connectionInfo, "dbadmin")
	if err != nil {
		log.Printf("Error opening database: %v", err)
		return err
	}
	defer db.Close()

	err = database.ResetDB(ctx, db)
	if err != nil {
		log.Printf("Error truncating database: %v", err)
	}
	return err
}
