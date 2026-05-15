package models

import (
	"database/sql"
	"os"
	"testing"
)

// newTestDB opens an in-memory SQLite database, applies schema.sql, and registers
// cleanup. The working directory for tests is the package directory, so schema.sql
// is resolved relative to backend/internal/models/.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("newTestDB: open: %v", err)
	}

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("newTestDB: read schema.sql: %v", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("newTestDB: exec schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// insertTestUser creates a user with the given username/password and returns their
// playerID. Fails the test immediately on error.
func insertTestUser(t *testing.T, m *UserModel, username, password string) int64 {
	t.Helper()
	id, err := m.InsertNew(username, password, &NewUserOptions{})
	if err != nil {
		t.Fatalf("insertTestUser(%q): %v", username, err)
	}
	return id
}
