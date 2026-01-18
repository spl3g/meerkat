package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func ConnectSQLite(dbName string) (*sql.DB, error) {
	return sql.Open("sqlite", fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=500&", dbName))
}
