package frontend

import (
	"testing"

	"github.com/tetsuo/fortune/internal/database"
)

var acquire func(*testing.T) (*database.DB, func())

func TestMain(m *testing.M) {
	database.RunDBTestsInParallel("fortune_mysql_test", 4, m, &acquire)
}
