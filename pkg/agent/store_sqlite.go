package agent

import (
	"database/sql"
	"fmt"
)

const sqliteBusyTimeoutMillis = 5000

func configureSQLiteDB(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("sqlite db is required")
	}

	// Allow a small pool because some store paths perform nested reads while a
	// parent query is still open. WAL + busy_timeout handle cross-process write
	// contention; a single connection causes self-deadlocks here.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		fmt.Sprintf("PRAGMA busy_timeout=%d;", sqliteBusyTimeoutMillis),
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to apply sqlite pragma %q: %w", pragma, err)
		}
	}

	return nil
}
