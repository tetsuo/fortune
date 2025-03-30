package database

import (
	"fmt"
)

// DBConfig holds the MySQL database configuration.
type DBConfig struct {
	DBHost     string `env:"DATABASE_HOST" envDefault:"localhost" json:"dbHost"`
	DBPort     string `env:"DATABASE_PORT" envDefault:"3306" json:"dbPort"`
	DBUser     string `env:"DATABASE_USER" envDefault:"root" json:"dbUser"`
	DBPassword string `env:"DATABASE_PASSWORD" envDefault:"example" json:"-"`
	DBName     string `env:"DATABASE_NAME" envDefault:"fortune_db" json:"dbName"`
}

// dataSourceName returns a MySQL connection DSN string for the given host.
func dataSourceName(c DBConfig, host string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		c.DBUser, c.DBPassword, host, c.DBPort, c.DBName,
	)
}

// DSN returns the primary database connection string.
func (c DBConfig) DSN() string {
	return dataSourceName(c, c.DBHost)
}
