package database

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

func ConnectSQLite(dbName string) (*sql.DB, error) {
	return sql.Open("sqlite", dbName)
}

