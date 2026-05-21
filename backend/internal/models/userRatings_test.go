package models

import (
	"testing"

	"burrchess/internal/chess"
	"golang.org/x/crypto/bcrypt"
)

func TestGetRatingTypeFromTimeFormat(t *testing.T) {
	// Boundaries: Bullet < 2min, Blitz < 5min, Rapid < 20min, Classical >= 20min
	tests := []struct {
		name   string
		ms     int64
		want   RatingType
	}{
		{"1 min (bullet)", 60_000, bullet},
		{"2 min boundary (blitz start)", chess.Bullet[1], blitz},
		{"3 min (blitz)", 3 * 60_000, blitz},
		{"5 min boundary (rapid start)", chess.Blitz[1], rapid},
		{"10 min (rapid)", 10 * 60_000, rapid},
		{"20 min boundary (classical start)", chess.Rapid[1], classical},
		{"30 min (classical)", 30 * 60_000, classical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRatingTypeFromTimeFormat(tt.ms)
			if got != tt.want {
				t.Errorf("GetRatingTypeFromTimeFormat(%d) = %v, want %v", tt.ms, got, tt.want)
			}
		})
	}
}

func TestUserRatings_GetRatingForTimeFormat(t *testing.T) {
	ratings := UserRatings{
		BulletRating:    1100,
		BlitzRating:     1200,
		RapidRating:     1300,
		ClassicalRating: 1400,
	}

	tests := []struct {
		name string
		ms   int64
		want int64
	}{
		{"bullet", 60_000, 1100},
		{"blitz", 3 * 60_000, 1200},
		{"rapid", 10 * 60_000, 1300},
		{"classical", 30 * 60_000, 1400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ratings.GetRatingForTimeFormat(tt.ms)
			if got != tt.want {
				t.Errorf("GetRatingForTimeFormat(%d) = %d, want %d", tt.ms, got, tt.want)
			}
		})
	}
}

func TestUserRatingsModel_DefaultRatings(t *testing.T) {
	db := newTestDB(t)
	users := &UserModel{DB: db, BcryptCost: bcrypt.MinCost}
	ratings := &UserRatingsModel{DB: db}

	insertTestUser(t, users, "grace", "pass")

	r, err := ratings.GetRatingFromUsername("grace")
	if err != nil {
		t.Fatalf("GetRatingFromUsername: %v", err)
	}
	if r.Username != "grace" {
		t.Errorf("got username=%q, want grace", r.Username)
	}
	for _, tc := range []struct {
		name string
		got  int64
	}{
		{"bullet", r.BulletRating},
		{"blitz", r.BlitzRating},
		{"rapid", r.RapidRating},
		{"classical", r.ClassicalRating},
	} {
		if tc.got != 1500 {
			t.Errorf("default %s rating = %d, want 1500", tc.name, tc.got)
		}
	}
}

func TestUserRatingsModel_GetRatingFromPlayerID(t *testing.T) {
	db := newTestDB(t)
	users := &UserModel{DB: db, BcryptCost: bcrypt.MinCost}
	ratingsModel := &UserRatingsModel{DB: db}

	id := insertTestUser(t, users, "henry", "pass")

	r, err := ratingsModel.GetRatingFromPlayerID(id)
	if err != nil {
		t.Fatalf("GetRatingFromPlayerID: %v", err)
	}
	if r.PlayerID != id {
		t.Errorf("got playerID=%d, want %d", r.PlayerID, id)
	}
}

func TestUserRatingsModel_UpdateRating(t *testing.T) {
	db := newTestDB(t)
	users := &UserModel{DB: db, BcryptCost: bcrypt.MinCost}
	ratingsModel := &UserRatingsModel{DB: db}

	id := insertTestUser(t, users, "iris", "pass")

	tests := []struct {
		name       string
		ratingType RatingType
		newRating  int64
	}{
		{"bullet", bullet, 1650},
		{"blitz", blitz, 1720},
		{"rapid", rapid, 1580},
		{"classical", classical, 1490},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ratingsModel.UpdateRatingFromPlayerID(id, tt.ratingType, tt.newRating); err != nil {
				t.Fatalf("UpdateRatingFromPlayerID: %v", err)
			}
		})
	}

	r, err := ratingsModel.GetRatingFromUsername("iris")
	if err != nil {
		t.Fatalf("GetRatingFromUsername: %v", err)
	}
	if r.BulletRating != 1650 {
		t.Errorf("BulletRating = %d, want 1650", r.BulletRating)
	}
	if r.BlitzRating != 1720 {
		t.Errorf("BlitzRating = %d, want 1720", r.BlitzRating)
	}
	if r.RapidRating != 1580 {
		t.Errorf("RapidRating = %d, want 1580", r.RapidRating)
	}
	if r.ClassicalRating != 1490 {
		t.Errorf("ClassicalRating = %d, want 1490", r.ClassicalRating)
	}
}

func TestUserRatingsModel_UpdateRatingFromUsername(t *testing.T) {
	db := newTestDB(t)
	users := &UserModel{DB: db, BcryptCost: bcrypt.MinCost}
	ratingsModel := &UserRatingsModel{DB: db}

	insertTestUser(t, users, "jack", "pass")

	if err := ratingsModel.UpdateRatingFromUsername("jack", blitz, 1800); err != nil {
		t.Fatalf("UpdateRatingFromUsername: %v", err)
	}

	r, err := ratingsModel.GetRatingFromUsername("jack")
	if err != nil {
		t.Fatalf("GetRatingFromUsername: %v", err)
	}
	if r.BlitzRating != 1800 {
		t.Errorf("BlitzRating = %d, want 1800", r.BlitzRating)
	}
}
