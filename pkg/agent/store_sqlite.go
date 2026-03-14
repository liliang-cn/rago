package agent

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
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
		fmt.Sprintf("PRAGMA busy_timeout=%d;", sqliteBusyTimeoutMillis),
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, pragma := range pragmas {
		if err := execSQLitePragmaWithRetry(db, pragma); err != nil {
			return fmt.Errorf("failed to apply sqlite pragma %q: %w", pragma, err)
		}
	}

	return nil
}

func execSQLitePragmaWithRetry(db *sql.DB, pragma string) error {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if _, err := db.Exec(pragma); err != nil {
			lastErr = err
			if isSQLiteLockError(err) {
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func isSQLiteLockError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy") || strings.Contains(msg, "locked (5)")
}
