package models

import (
	"database/sql"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"", true},                         // empty = remove email, allowed
		{"user@example.com", true},
		{"user@sub.example.com", true},
		{"u@a.io", true},
		{"notanemail", false},
		{"@example.com", false},            // nothing before @
		{"user@", false},                   // nothing after @
		{"user@nodot", false},              // no dot in domain
		{"user@domain.", false},            // trailing dot
		{"user@@domain.com", false},        // double @
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := IsValidEmail(tt.email); got != tt.valid {
				t.Errorf("IsValidEmail(%q) = %v, want %v", tt.email, got, tt.valid)
			}
		})
	}
}

func TestGenerateNewPlayerId(t *testing.T) {
	for range 100 {
		id := GenerateNewPlayerId()
		if id <= 0 {
			t.Errorf("GenerateNewPlayerId() = %d, want positive int64", id)
		}
	}
}

func TestUserModel_InsertNew(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}

	id, err := m.InsertNew("alice", "password123", &NewUserOptions{})
	if err != nil {
		t.Fatalf("InsertNew: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive playerID, got %d", id)
	}
}

func TestUserModel_InsertNew_DuplicateUsername(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}

	if _, err := m.InsertNew("alice", "pass1", &NewUserOptions{}); err != nil {
		t.Fatalf("first InsertNew: %v", err)
	}

	_, err := m.InsertNew("alice", "pass2", &NewUserOptions{})
	if err == nil {
		t.Error("expected error on duplicate username, got nil")
	}
}

func TestUserModel_Authenticate(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}
	insertedID := insertTestUser(t, m, "bob", "secret")

	t.Run("correct password", func(t *testing.T) {
		id, ok := m.Authenticate("bob", "secret")
		if !ok {
			t.Fatal("expected authorized=true, got false")
		}
		if id != insertedID {
			t.Errorf("got playerID=%d, want %d", id, insertedID)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		_, ok := m.Authenticate("bob", "wrongpassword")
		if ok {
			t.Error("expected authorized=false for wrong password")
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		_, ok := m.Authenticate("nobody", "secret")
		if ok {
			t.Error("expected authorized=false for unknown user")
		}
	})
}

func TestUserModel_GetUserClientSide(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}
	id := insertTestUser(t, m, "carol", "pass")

	t.Run("by username", func(t *testing.T) {
		u, err := m.GetUserClientSide(ByUsername("carol"))
		if err != nil {
			t.Fatalf("GetUserClientSide: %v", err)
		}
		if u.Username != "carol" {
			t.Errorf("got username=%q, want carol", u.Username)
		}
		if u.PlayerID != id {
			t.Errorf("got playerID=%d, want %d", u.PlayerID, id)
		}
	})

	t.Run("by playerID", func(t *testing.T) {
		u, err := m.GetUserClientSide(ByPlayerID(id))
		if err != nil {
			t.Fatalf("GetUserClientSide: %v", err)
		}
		if u.Username != "carol" {
			t.Errorf("got username=%q, want carol", u.Username)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := m.GetUserClientSide(ByUsername("nobody"))
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestUserModel_SearchForUsers(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}
	for _, name := range []string{"alice", "alicia", "bob"} {
		insertTestUser(t, m, name, "p")
	}

	t.Run("prefix match", func(t *testing.T) {
		results, err := m.SearchForUsers("ali")
		if err != nil {
			t.Fatalf("SearchForUsers: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})

	t.Run("exact prefix", func(t *testing.T) {
		results, err := m.SearchForUsers("bob")
		if err != nil {
			t.Fatalf("SearchForUsers: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		results, err := m.SearchForUsers("ALI")
		if err != nil {
			t.Fatalf("SearchForUsers: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})

	t.Run("glob wildcard escaped", func(t *testing.T) {
		// A bare '*' must not match every user — it should be escaped.
		results, err := m.SearchForUsers("*")
		if err != nil {
			t.Fatalf("SearchForUsers: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("'*' should be escaped and match nothing, got %d results", len(results))
		}
	})

	t.Run("no match", func(t *testing.T) {
		results, err := m.SearchForUsers("xyz")
		if err != nil {
			t.Fatalf("SearchForUsers: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})
}

func TestUserModel_UpdateEmail(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}
	id := insertTestUser(t, m, "dave", "pass")

	if err := m.UpdateEmail(id, "dave@example.com"); err != nil {
		t.Fatalf("UpdateEmail: %v", err)
	}

	settings, err := m.GetUserAccountSettings(id)
	if err != nil {
		t.Fatalf("GetUserAccountSettings: %v", err)
	}
	if settings.Email == nil || *settings.Email != "dave@example.com" {
		t.Errorf("got email=%v, want dave@example.com", settings.Email)
	}

	// Clear email by passing empty string
	if err := m.UpdateEmail(id, ""); err != nil {
		t.Fatalf("UpdateEmail (clear): %v", err)
	}
	settings, err = m.GetUserAccountSettings(id)
	if err != nil {
		t.Fatalf("GetUserAccountSettings after clear: %v", err)
	}
	if settings.Email != nil {
		t.Errorf("expected nil email after clearing, got %q", *settings.Email)
	}
}

func TestUserModel_UpdatePassword(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}
	id := insertTestUser(t, m, "eve", "oldpass")

	if err := m.UpdatePassword(id, "newpass"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	_, ok := m.Authenticate("eve", "newpass")
	if !ok {
		t.Error("expected auth with new password to succeed")
	}
	_, ok = m.Authenticate("eve", "oldpass")
	if ok {
		t.Error("expected auth with old password to fail after update")
	}
}

func TestUserModel_GetTileInfoFromUsername(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}
	insertTestUser(t, m, "frank", "pass")

	info, err := m.GetTileInfoFromUsername("frank")
	if err != nil {
		t.Fatalf("GetTileInfoFromUsername: %v", err)
	}
	if info.Username != "frank" {
		t.Errorf("got username=%q, want frank", info.Username)
	}
	// Default ratings are 1500
	if info.Ratings.BulletRating != 1500 {
		t.Errorf("BulletRating = %d, want 1500", info.Ratings.BulletRating)
	}
	if info.Ratings.BlitzRating != 1500 {
		t.Errorf("BlitzRating = %d, want 1500", info.Ratings.BlitzRating)
	}
	if info.NumberOfGames != 0 {
		t.Errorf("NumberOfGames = %d, want 0", info.NumberOfGames)
	}
}

func TestUserModel_GetUserServerSide(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}
	insertTestUser(t, m, "grace", "pass")

	u, err := m.GetUserServerSide(ByUsername("grace"))
	if err != nil {
		t.Fatalf("GetUserServerSide: %v", err)
	}
	if u.Username != "grace" {
		t.Errorf("username = %q, want grace", u.Username)
	}
	if u.PlayerID == 0 {
		t.Error("PlayerID should be non-zero")
	}
	// Password hash should be present
	if len(u.Password) == 0 {
		t.Error("expected non-empty password hash")
	}
}

func TestUserModel_GetUserServerSide_NotFound(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}

	_, err := m.GetUserServerSide(ByUsername("nobody"))
	if err == nil {
		t.Error("expected error for unknown user, got nil")
	}
}

func TestUserModel_UpdateLastSeen(t *testing.T) {
	m := &UserModel{DB: newTestDB(t), BcryptCost: bcrypt.MinCost}
	insertTestUser(t, m, "henry", "pass")

	// Capture last_seen before the update.
	before, err := m.GetUserServerSide(ByUsername("henry"))
	if err != nil {
		t.Fatalf("get before: %v", err)
	}

	if err := m.UpdateLastSeen(ByUsername("henry")); err != nil {
		t.Fatalf("UpdateLastSeen: %v", err)
	}

	after, err := m.GetUserServerSide(ByUsername("henry"))
	if err != nil {
		t.Fatalf("get after: %v", err)
	}

	// last_seen should be >= what it was before (SQLite stores unix seconds).
	if after.LastSeen < before.LastSeen {
		t.Errorf("LastSeen went backwards: before=%d after=%d", before.LastSeen, after.LastSeen)
	}
}
