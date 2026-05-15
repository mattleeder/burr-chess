package main

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// getKFactor
// ---------------------------------------------------------------------------

func TestGetKFactor(t *testing.T) {
	tests := []struct {
		elo  int64
		want float64
	}{
		{0, 32},
		{1500, 32},
		{2099, 32},
		{2100, 24}, // boundary: inclusive lower bound of middle tier
		{2200, 24},
		{2400, 24}, // boundary: inclusive upper bound of middle tier
		{2401, 16},
		{3000, 16},
	}

	for _, tt := range tests {
		got := getKFactor(tt.elo)
		if got != tt.want {
			t.Errorf("getKFactor(%d) = %v, want %v", tt.elo, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// calculateEloChanges
// ---------------------------------------------------------------------------

func TestCalculateEloChanges_EqualEloWhiteWins(t *testing.T) {
	// Both players at 1500 → K=32, expected score = 0.5 each.
	// White wins (1, 0): white +16, black −16.
	whiteGain, blackGain := calculateEloChanges(1500, 1.0, 1500, 0.0)

	if whiteGain != 16.0 {
		t.Errorf("whiteGain = %v, want 16", whiteGain)
	}
	if blackGain != -16.0 {
		t.Errorf("blackGain = %v, want -16", blackGain)
	}
}

func TestCalculateEloChanges_EqualEloDraw(t *testing.T) {
	// Draw against equal opponent → no rating change.
	whiteGain, blackGain := calculateEloChanges(1500, 0.5, 1500, 0.5)

	if whiteGain != 0.0 {
		t.Errorf("whiteGain = %v, want 0", whiteGain)
	}
	if blackGain != 0.0 {
		t.Errorf("blackGain = %v, want 0", blackGain)
	}
}

func TestCalculateEloChanges_EqualEloBlackWins(t *testing.T) {
	whiteGain, blackGain := calculateEloChanges(1500, 0.0, 1500, 1.0)

	if whiteGain != -16.0 {
		t.Errorf("whiteGain = %v, want -16", whiteGain)
	}
	if blackGain != 16.0 {
		t.Errorf("blackGain = %v, want 16", blackGain)
	}
}

func TestCalculateEloChanges_UpsetWin(t *testing.T) {
	// Lower-rated player (1200) beats higher-rated (1800).
	// Lower-rated has a much higher expected gain.
	loserGain, winnerGain := calculateEloChanges(1800, 0.0, 1200, 1.0)

	if winnerGain <= 0 {
		t.Errorf("upset winner should gain rating, got %v", winnerGain)
	}
	if loserGain >= 0 {
		t.Errorf("upset loser should lose rating, got %v", loserGain)
	}
	// The underdog gains more than the favourite would have gained
	favouriteWin, _ := calculateEloChanges(1800, 1.0, 1200, 0.0)
	if winnerGain <= favouriteWin {
		t.Errorf("underdog win gain (%v) should exceed favourite win gain (%v)", winnerGain, favouriteWin)
	}
}

func TestCalculateEloChanges_ZeroSumWithSameKFactor(t *testing.T) {
	// With identical K-factors the gains must sum to zero.
	tests := []struct {
		elo1, elo2   int64
		p1pts, p2pts float64
	}{
		{1500, 1500, 1.0, 0.0},
		{1500, 1500, 0.5, 0.5},
		{1000, 2000, 1.0, 0.0}, // both < 2100 → K=32
	}
	for _, tt := range tests {
		g1, g2 := calculateEloChanges(tt.elo1, tt.p1pts, tt.elo2, tt.p2pts)
		sum := g1 + g2
		if math.Abs(sum) > 1e-9 {
			t.Errorf("elo changes not zero-sum: %v+%v=%v (elos %d/%d)", g1, g2, sum, tt.elo1, tt.elo2)
		}
	}
}

func TestCalculateEloChanges_FavouriteWinsLittleGain(t *testing.T) {
	// A heavily favoured player (2000 vs 1000) should gain very little on a win.
	gain, _ := calculateEloChanges(2000, 1.0, 1000, 0.0)
	if gain >= 5.0 {
		t.Errorf("heavily favoured winner should gain < 5 points, got %v", gain)
	}
	if gain < 0 {
		t.Errorf("winner should not lose rating, got %v", gain)
	}
}
