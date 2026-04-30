package chess

import (
	"slices"
	"sort"
	"testing"
)

func TestGetValidMovesForPiece(t *testing.T) {
	// Test table
	tests := []struct {
		name                                     string
		fen                                      string
		pieceIndex                               int
		expectedMoves                            []int
		expectedCaptures                         []int
		expectedTriggerPromotion                 bool
		expectedFriendlyKingInCheckOrPiecePinned bool
	}{
		// Starting position

		// Pawns
		{
			name:          "starting position pawn a2",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    48,
			expectedMoves: []int{40, 32},
		},
		{
			name:          "starting position pawn b2",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    49,
			expectedMoves: []int{41, 33},
		},
		{
			name:          "starting position pawn c2",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    50,
			expectedMoves: []int{42, 34},
		},
		{
			name:          "starting position pawn d2",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    51,
			expectedMoves: []int{43, 35},
		},
		{
			name:          "starting position pawn e2",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    52,
			expectedMoves: []int{44, 36},
		},
		{
			name:          "starting position pawn f2",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    53,
			expectedMoves: []int{45, 37},
		},
		{
			name:          "starting position pawn g2",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    54,
			expectedMoves: []int{46, 38},
		},
		{
			name:          "starting position pawn h2",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    55,
			expectedMoves: []int{47, 39},
		},

		// Knights
		{
			name:          "starting position knight b1",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    57,
			expectedMoves: []int{40, 42},
		},
		{
			name:          "starting position knight g1",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    62,
			expectedMoves: []int{45, 47},
		},

		// Blocked pieces — no legal moves
		{
			name:          "starting position rook a1",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    56,
			expectedMoves: []int{},
		},
		{
			name:          "starting position bishop c1",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    58,
			expectedMoves: []int{},
		},
		{
			name:          "starting position queen d1",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    59,
			expectedMoves: []int{},
		},
		{
			name:          "starting position king e1",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    60,
			expectedMoves: []int{},
		},
		{
			name:          "starting position bishop f1",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    61,
			expectedMoves: []int{},
		},
		{
			name:          "starting position rook h1",
			fen:           "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:    63,
			expectedMoves: []int{},
		},

		// === WHITE EN PASSANT ===

		// Capture left
		{
			name:             "white en passant capture left e5xd6",
			fen:              "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1",
			pieceIndex:       28,        // e5
			expectedMoves:    []int{20}, // e6
			expectedCaptures: []int{19}, // d6 en passant
		},
		// Capture right
		{
			name:             "white en passant capture right d5xe6",
			fen:              "4k3/8/8/3Pp3/8/8/8/4K3 w - e6 0 1",
			pieceIndex:       27,        // d5
			expectedMoves:    []int{19}, // d6
			expectedCaptures: []int{20}, // e6 en passant
		},
		// Edge file a captures b6
		{
			name:             "white en passant a-file a5xb6",
			fen:              "4k3/8/8/Pp6/8/8/8/4K3 w - b6 0 1",
			pieceIndex:       24,        // a5
			expectedMoves:    []int{16}, // a6
			expectedCaptures: []int{17}, // b6 en passant
		},
		// Edge file h captures g6
		{
			name:             "white en passant h-file h5xg6",
			fen:              "4k3/8/8/6pP/8/8/8/4K3 w - g6 0 1",
			pieceIndex:       31,        // h5
			expectedMoves:    []int{23}, // h6
			expectedCaptures: []int{22}, // g6 en passant
		},
		// All files: b5 captures a6
		{
			name:             "white en passant b5xa6",
			fen:              "4k3/8/8/pP6/8/8/8/4K3 w - a6 0 1",
			pieceIndex:       25,        // b5
			expectedMoves:    []int{17}, // b6
			expectedCaptures: []int{16}, // a6 en passant
		},
		// All files: c5 captures d6
		{
			name:             "white en passant c5xd6",
			fen:              "4k3/8/8/2Pp4/8/8/8/4K3 w - d6 0 1",
			pieceIndex:       26,        // c5
			expectedMoves:    []int{18}, // c6
			expectedCaptures: []int{19}, // d6 en passant
		},
		// All files: f5 captures g6
		{
			name:             "white en passant f5xg6",
			fen:              "4k3/8/8/5Pp1/8/8/8/4K3 w - g6 0 1",
			pieceIndex:       29,        // f5
			expectedMoves:    []int{21}, // f6
			expectedCaptures: []int{22}, // g6 en passant
		},
		// All files: g5 captures h6
		{
			name:             "white en passant g5xh6",
			fen:              "4k3/8/8/6Pp/8/8/8/4K3 w - h6 0 1",
			pieceIndex:       30,        // g5
			expectedMoves:    []int{22}, // g6
			expectedCaptures: []int{23}, // h6 en passant
		},
		// En passant + normal capture on the other diagonal
		{
			name:             "white en passant left plus normal capture right",
			fen:              "4k3/8/5n2/3pP3/8/8/8/4K3 w - d6 0 1",
			pieceIndex:       28,            // e5
			expectedMoves:    []int{20},     // e6
			expectedCaptures: []int{19, 21}, // d6 en passant + f6 normal capture
		},
		// Adjacent pawn but NO en passant square — should not be capturable
		{
			name:             "white adjacent pawn but no en passant flag",
			fen:              "4k3/8/8/3pP3/8/8/8/4K3 w - - 0 1",
			pieceIndex:       28,        // e5
			expectedMoves:    []int{20}, // e6 only
			expectedCaptures: []int{},
		},

		// === BLACK EN PASSANT ===

		// Capture left
		{
			name:             "black en passant capture left e4xd3",
			fen:              "4k3/8/8/8/3Pp3/8/8/4K3 b - d3 0 1",
			pieceIndex:       36,        // e4
			expectedMoves:    []int{44}, // e3
			expectedCaptures: []int{43}, // d3 en passant
		},
		// Capture right
		{
			name:             "black en passant capture right d4xe3",
			fen:              "4k3/8/8/8/3pP3/8/8/4K3 b - e3 0 1",
			pieceIndex:       35,        // d4
			expectedMoves:    []int{43}, // d3
			expectedCaptures: []int{44}, // e3 en passant
		},
		// Edge file a captures b3
		{
			name:             "black en passant a-file a4xb3",
			fen:              "4k3/8/8/8/pP6/8/8/4K3 b - b3 0 1",
			pieceIndex:       32,        // a4
			expectedMoves:    []int{40}, // a3
			expectedCaptures: []int{41}, // b3 en passant
		},
		// Edge file h captures g3
		{
			name:             "black en passant h-file h4xg3",
			fen:              "4k3/8/8/8/6Pp/8/8/4K3 b - g3 0 1",
			pieceIndex:       39,        // h4
			expectedMoves:    []int{47}, // h3
			expectedCaptures: []int{46}, // g3 en passant
		},

		// === EDGE CASE: EN PASSANT EXPOSES KING (HORIZONTAL PIN) ===

		// King on a5, white pawn d5, black pawn e5 (EP), black rook h5
		// Taking en passant removes both pawns from rank 5, exposing king to rook
		{
			name:             "white en passant illegal — exposes king to horizontal rook",
			fen:              "4k3/8/8/K2Pp2r/8/8/8/8 w - e6 0 1",
			pieceIndex:       27,        // d5
			expectedMoves:    []int{19}, // d6 forward only
			expectedCaptures: []int{},   // e6 en passant blocked by pin
		},

		{
			name:             "white en passant illegal — exposes king to bishop",
			fen:              "8/8/6K1/4Pp2/8/8/2b5/k7 w - - 0 1",
			pieceIndex:       28,        // e5
			expectedMoves:    []int{20}, // e6 forward only
			expectedCaptures: []int{},   // f6 en passant blocked by bishop pin
		},

		// ============================================================
		// PIECE MOVEMENT — each piece on an open board
		// ============================================================

		// --- Pawn ---
		{
			name:             "white pawn e4 single push",
			fen:              "4k3/8/8/8/4P3/8/8/4K3 w - - 0 1",
			pieceIndex:       36,        // e4
			expectedMoves:    []int{28}, // e5
			expectedCaptures: []int{},
		},
		{
			name:             "white pawn e2 double push from start",
			fen:              "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
			pieceIndex:       52,            // e2
			expectedMoves:    []int{44, 36}, // e3, e4
			expectedCaptures: []int{},
		},
		{
			name:             "white pawn captures both diagonals",
			fen:              "4k3/8/8/8/8/3p1p2/4P3/4K3 w - - 0 1",
			pieceIndex:       52,            // e2
			expectedMoves:    []int{44, 36}, // e3, e4
			expectedCaptures: []int{43, 45}, // d3, f3
		},
		{
			name:             "white pawn blocked by piece ahead",
			fen:              "4k3/8/8/8/8/4p3/4P3/4K3 w - - 0 1",
			pieceIndex:       52, // e2
			expectedMoves:    []int{},
			expectedCaptures: []int{},
		},
		{
			name:             "white pawn double push blocked on 4th rank",
			fen:              "4k3/8/8/8/4p3/8/4P3/4K3 w - - 0 1",
			pieceIndex:       52,        // e2
			expectedMoves:    []int{44}, // e3 only, e4 blocked
			expectedCaptures: []int{},
		},
		{
			name:             "black pawn e7 double push from start",
			fen:              "4k3/4p3/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:       12,            // e7
			expectedMoves:    []int{20, 28}, // e6, e5
			expectedCaptures: []int{},
		},
		{
			name:             "black pawn e5 single push",
			fen:              "4k3/8/8/4p3/8/8/8/4K3 b - - 0 1",
			pieceIndex:       28,        // e5
			expectedMoves:    []int{36}, // e4
			expectedCaptures: []int{},
		},
		{
			name:             "black pawn captures both diagonals",
			fen:              "4k3/4p3/3P1P2/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:       12,            // e7
			expectedMoves:    []int{20, 28}, // e6, e5
			expectedCaptures: []int{19, 21}, // d6, f6
		},
		{
			name:             "white pawn a-file can only capture right",
			fen:              "4k3/8/8/8/8/1p6/P7/4K3 w - - 0 1",
			pieceIndex:       48,            // a2
			expectedMoves:    []int{40, 32}, // a3, a4
			expectedCaptures: []int{41},     // b3 only
		},
		{
			name:             "white pawn h-file can only capture left",
			fen:              "4k3/8/8/8/8/6p1/7P/4K3 w - - 0 1",
			pieceIndex:       55,            // h2
			expectedMoves:    []int{47, 39}, // h3, h4
			expectedCaptures: []int{46},     // g3 only
		},

		// --- Knight ---
		{
			name:             "knight center e4 all 8 squares",
			fen:              "4k3/8/8/8/4N3/8/8/4K3 w - - 0 1",
			pieceIndex:       36,                                    // e4
			expectedMoves:    []int{21, 30, 46, 53, 51, 42, 26, 19}, // f6,g5,g3,f2,d2,c3,c5,d6
			expectedCaptures: []int{},
		},
		{
			name:             "knight corner a1 only 2 squares",
			fen:              "4k3/8/8/8/8/8/8/N3K3 w - - 0 1",
			pieceIndex:       56,            // a1
			expectedMoves:    []int{41, 50}, // b3, c2
			expectedCaptures: []int{},
		},
		{
			name:             "knight corner h1 only 2 squares",
			fen:              "4k3/8/8/8/8/8/8/4K2N w - - 0 1",
			pieceIndex:       63,            // h1
			expectedMoves:    []int{46, 53}, // g3, f2
			expectedCaptures: []int{},
		},
		{
			name:             "knight corner a8 only 2 squares",
			fen:              "N3k3/8/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:       0,             // a8
			expectedMoves:    []int{10, 17}, // c7, b6
			expectedCaptures: []int{},
		},
		{
			name:             "knight edge b1 limited moves",
			fen:              "4k3/8/8/8/8/8/8/1N2K3 w - - 0 1",
			pieceIndex:       57,                // b1
			expectedMoves:    []int{40, 42, 51}, // a3, c3, d2
			expectedCaptures: []int{},
		},
		{
			name:             "knight captures enemy pieces",
			fen:              "4k3/8/3p1p2/8/4N3/8/8/4K3 w - - 0 1",
			pieceIndex:       36,                            // e4
			expectedMoves:    []int{30, 46, 53, 51, 42, 26}, // g5,g3,f2,d2,c3,c5
			expectedCaptures: []int{19, 21},                 // d6, f6
		},
		{
			name:             "knight blocked by friendly pieces",
			fen:              "4k3/8/3P1P2/8/4N3/8/8/4K3 w - - 0 1",
			pieceIndex:       36,                            // e4
			expectedMoves:    []int{30, 46, 53, 51, 42, 26}, // g5,g3,f2,d2,c3,c5
			expectedCaptures: []int{},                       // d6,f6 are friendly
		},

		// --- Bishop ---
		{
			name:       "bishop center d4 open board",
			fen:        "4k3/8/8/8/3B4/8/8/4K3 w - - 0 1",
			pieceIndex: 35, // d4
			expectedMoves: []int{
				26, 17, 8, // c5, b6, a7 (up-left, dir -9)
				28, 21, 14, 7, // e5, f6, g7, h8 (up-right, dir -7)
				42, 49, 56, // c3, b2, a1 (down-left, dir +7)
				44, 53, 62, // e3, f2, g1 (down-right, dir +9)
			},
			expectedCaptures: []int{},
		},
		{
			name:             "bishop corner a1",
			fen:              "4k3/8/8/8/8/8/8/B3K3 w - - 0 1",
			pieceIndex:       56,                               // a1
			expectedMoves:    []int{49, 42, 35, 28, 21, 14, 7}, // b2 through h8
			expectedCaptures: []int{},
		},
		{
			name:             "bishop blocked by friendly pieces",
			fen:              "4k3/8/8/2P1P3/3B4/2P1P3/8/4K3 w - - 0 1",
			pieceIndex:       35,      // d4
			expectedMoves:    []int{}, // all diagonals blocked by friendly pawns
			expectedCaptures: []int{},
		},
		{
			name:             "bishop captures and stops",
			fen:              "4k3/8/8/2p1p3/3B4/2p1p3/8/4K3 w - - 0 1",
			pieceIndex:       35, // d4
			expectedMoves:    []int{},
			expectedCaptures: []int{26, 28, 42, 44}, // c5, e5, c3, e3
		},

		// --- Rook ---
		{
			name:       "rook center d4 open board",
			fen:        "4k3/8/8/8/3R4/8/8/4K3 w - - 0 1",
			pieceIndex: 35, // d4
			expectedMoves: []int{
				27, 19, 11, 3, // d5, d6, d7, d8 (up, dir -8)
				43, 51, 59, // d3, d2, d1 (down, dir +8)
				34, 33, 32, // c4, b4, a4 (left, dir -1)
				36, 37, 38, 39, // e4, f4, g4, h4 (right, dir +1)
			},
			expectedCaptures: []int{},
		},
		{
			name:             "rook blocked by friendly pieces",
			fen:              "4k3/8/8/3P4/2PRP3/3P4/8/4K3 w - - 0 1",
			pieceIndex:       35, // d4
			expectedMoves:    []int{},
			expectedCaptures: []int{},
		},
		{
			name:             "rook captures on each axis",
			fen:              "4k3/8/8/3p4/2pRp3/3p4/8/4K3 w - - 0 1",
			pieceIndex:       35,
			expectedMoves:    []int{},
			expectedCaptures: []int{27, 43, 34, 36}, // d5, d3, c4, e4
		},
		{
			name:       "rook a1 corner open",
			fen:        "4k3/8/8/8/8/8/8/R3K3 w - - 0 1",
			pieceIndex: 56, // a1
			expectedMoves: []int{
				48, 40, 32, 24, 16, 8, 0, // a2-a8 (up)
				57, 58, 59, // b1, c1, d1 (right, blocked by king on e1)
			},
			expectedCaptures: []int{},
		},

		// --- Queen ---
		{
			name:       "queen center d4 open board",
			fen:        "4k3/8/8/8/3Q4/8/8/4K3 w - - 0 1",
			pieceIndex: 35,
			expectedMoves: []int{
				// Diagonals
				26, 17, 8, // up-left
				28, 21, 14, 7, // up-right
				42, 49, 56, // down-left
				44, 53, 62, // down-right
				// Straights
				27, 19, 11, 3, // up
				43, 51, 59, // down
				34, 33, 32, // left
				36, 37, 38, 39, // right
			},
			expectedCaptures: []int{},
		},
		{
			name:             "queen surrounded by friendly pieces",
			fen:              "4k3/8/8/2PPP3/2PQP3/2PPP3/8/4K3 w - - 0 1",
			pieceIndex:       35,
			expectedMoves:    []int{},
			expectedCaptures: []int{},
		},
		{
			name:             "queen surrounded by enemy pieces",
			fen:              "4k3/8/8/2ppp3/2pQp3/2ppp3/8/4K3 w - - 0 1",
			pieceIndex:       35,
			expectedMoves:    []int{},
			expectedCaptures: []int{26, 27, 28, 34, 36, 42, 43, 44},
		},

		// --- King ---
		{
			name:             "king center e4 all 8 squares",
			fen:              "4k3/8/8/8/4K3/8/8/8 w - - 0 1",
			pieceIndex:       36, // e4
			expectedMoves:    []int{27, 28, 29, 35, 37, 43, 44, 45},
			expectedCaptures: []int{},
		},
		{
			name:             "king corner a1",
			fen:              "4k3/8/8/8/8/8/8/K7 w - - 0 1",
			pieceIndex:       56,
			expectedMoves:    []int{48, 49, 57},
			expectedCaptures: []int{},
		},
		{
			name:             "king captures adjacent enemy pieces",
			fen:              "4k3/8/8/8/3pKp2/8/8/8 w - - 0 1",
			pieceIndex:       36,
			expectedMoves:    []int{27, 28, 29, 43, 45}, // d5, e5, f5, d3, f3, e3 is blocked by pawns
			expectedCaptures: []int{35, 37},             // d4, f4
		},
		{
			name:             "king blocked by friendly pieces",
			fen:              "4k3/8/8/3PPP2/3PKP2/3PPP2/8/8 w - - 0 1",
			pieceIndex:       36,
			expectedMoves:    []int{},
			expectedCaptures: []int{},
		},

		// ============================================================
		// CASTLING
		// ============================================================

		// White full castling
		{
			name:             "white full castling rights available",
			fen:              "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			pieceIndex:       60,                    // e1
			expectedMoves:    []int{58, 59, 61, 62}, // c1, d1, f1, g1 (c1 = queenside castle, g1 = kingside castle)
			expectedCaptures: []int{},
		},
		// White queenside
		{
			name:       "white queenside castle available",
			fen:        "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w Qkq - 0 1",
			pieceIndex: 60,
			// d1=59, f1=61, c1=58(queenside)
			expectedMoves:    []int{58, 59, 61},
			expectedCaptures: []int{},
		},
		// White kingside
		{
			name:       "white kingside castle available",
			fen:        "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w Kkq - 0 1",
			pieceIndex: 60,
			// d1=59, f1=61, g1=62(kingside)
			expectedMoves:    []int{59, 61, 62},
			expectedCaptures: []int{},
		},
		// White castle rights revoked
		{
			name:             "white no castling rights",
			fen:              "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w - - 0 1",
			pieceIndex:       60,
			expectedMoves:    []int{59, 61}, // d1, f1 only
			expectedCaptures: []int{},
		},
		// Black full castle
		{
			name:             "black full castling rights available",
			fen:              "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQkq - 0 1",
			pieceIndex:       4,                 // e8
			expectedMoves:    []int{2, 3, 5, 6}, // c2 (castle), d8, f8, g8(castle)
			expectedCaptures: []int{},
		},
		// Black kingside castle
		{
			name:             "black kingside castle available",
			fen:              "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQk - 0 1",
			pieceIndex:       4,              // e8
			expectedMoves:    []int{3, 5, 6}, // d8, f8, g8(castle)
			expectedCaptures: []int{},
		},
		// Black queenside castle
		{
			name:             "black queenside available",
			fen:              "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQq - 0 1",
			pieceIndex:       4,
			expectedMoves:    []int{2, 3, 5}, // c8 (castle), d8, f8, g8
			expectedCaptures: []int{},
		},
		// Castle blocked by piece in the way
		{
			name:             "white kingside castle blocked by bishop",
			fen:              "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K1BR w KQkq - 0 1",
			pieceIndex:       60,
			expectedMoves:    []int{58, 59, 61}, // c1 (castle), d1, f1
			expectedCaptures: []int{},
		},
		{
			name:             "white queenside castle blocked by bishop",
			fen:              "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/RB2K2R w KQkq - 0 1",
			pieceIndex:       60,
			expectedMoves:    []int{59, 61, 62}, // d1, f1, g1(castle), queenside castle blocked
			expectedCaptures: []int{},
		},
		// Castle through check — king passes through attacked square
		{
			name: "white kingside castle blocked by attack on f1",
			// Black rook on f8 attacks f1 through the file
			fen:        "4kr2/8/8/8/8/8/PPPPP1PP/R3K2R w KQ - 0 1",
			pieceIndex: 60,
			// f1(61) is attacked by rook on f8, can't castle through it
			// But king can still go to d1
			expectedMoves:    []int{59, 58}, // d1, c1 (castle)
			expectedCaptures: []int{},
		},
		// Castle while in check — not allowed
		{
			name:       "white cannot castle while in check",
			fen:        "4k3/8/8/8/4r3/8/PPPP1PPP/R3K2R w KQ - 0 1",
			pieceIndex: 60,
			// King is in check from rook on e4. Must block or move.
			// King can go to d1(59), f1(61), d2(51) — but need to verify which are safe
			expectedMoves:                            []int{59, 61}, // d1, f1 (flee check)
			expectedCaptures:                         []int{},
			expectedTriggerPromotion:                 false,
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// ============================================================
		// KING IN CHECK — must block, capture, or flee
		// ============================================================

		{
			name:       "king in check from rook — must move",
			fen:        "4k3/8/8/8/8/8/8/r3K3 w - - 0 1",
			pieceIndex: 60, // e1
			// Rook on d1(59) gives check. King must flee.
			expectedMoves:                            []int{51, 52, 53},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:       "king in check — can capture the attacker",
			fen:        "4k3/8/8/8/8/8/8/3rK3 w - - 0 1",
			pieceIndex: 60, // e1
			// Rook on d1(59) gives check. King can capture d1 or flee.
			expectedMoves:                            []int{52, 53},
			expectedCaptures:                         []int{59}, // capture rook on d1
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name: "pawn can block check",
			// Rook on d1(59) gives check. Pawn must block.
			fen:                                      "4k3/8/8/8/3PPP2/r3K3/3P3r/8 w - - 0 1",
			pieceIndex:                               51,
			expectedMoves:                            []int{43},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// ============================================================
		// PROMOTION
		// ============================================================

		// --- White promotion ---
		{
			name:                     "white pawn promotes by pushing",
			fen:                      "4k3/4P3/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               12,      // e7
			expectedMoves:            []int{}, // e8
			expectedCaptures:         []int{},
			expectedTriggerPromotion: true,
		},
		{
			name:                     "white pawn promotes by capturing",
			fen:                      "3rkr2/4P3/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               12,          // e7
			expectedMoves:            []int{},     // e8 blocked by king
			expectedCaptures:         []int{3, 5}, // capture d8 rook, f8 rook
			expectedTriggerPromotion: true,
		},
		{
			name:                     "white pawn promotes push and capture both available",
			fen:                      "2kn4/4P3/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               12,       // e7
			expectedMoves:            []int{4}, // e8
			expectedCaptures:         []int{3}, // capture d8 knight
			expectedTriggerPromotion: true,
		},
		{
			name:                     "white pawn promotes on a-file push",
			fen:                      "4k3/P7/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               8,        // a7
			expectedMoves:            []int{0}, // a8
			expectedCaptures:         []int{},
			expectedTriggerPromotion: true,
		},
		{
			name:                     "white pawn promotes on h-file push",
			fen:                      "4k3/7P/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               15,       // h7
			expectedMoves:            []int{7}, // h8
			expectedCaptures:         []int{},
			expectedTriggerPromotion: true,
		},
		{
			name:                     "white pawn promotes on a-file capture only right",
			fen:                      "1n2k3/P7/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               8,        // a7
			expectedMoves:            []int{0}, // a8
			expectedCaptures:         []int{1}, // capture b8 knight
			expectedTriggerPromotion: true,
		},
		{
			name:                     "white pawn promotes on h-file capture only left",
			fen:                      "4k1n1/7P/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               15,       // h7
			expectedMoves:            []int{7}, // h8
			expectedCaptures:         []int{6}, // capture g8 knight
			expectedTriggerPromotion: true,
		},
		{
			name:                     "white pawn blocked on 7th rank no promotion",
			fen:                      "4kn2/5P2/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               13,      // f7
			expectedMoves:            []int{}, // f8 blocked by knight
			expectedCaptures:         []int{},
			expectedTriggerPromotion: true, // flag is still set based on position, even with no moves
		},

		// --- Black promotion ---
		{
			name:                     "black pawn promotes by pushing",
			fen:                      "4k3/8/8/8/8/8/4p3/2K5 b - - 0 1",
			pieceIndex:               52,        // e2
			expectedMoves:            []int{60}, // e1
			expectedCaptures:         []int{},
			expectedTriggerPromotion: true,
		},
		{
			name:                     "black pawn promotes by capturing",
			fen:                      "4k3/8/8/8/8/8/4p3/3RKR2 b - - 0 1",
			pieceIndex:               52,            // e2
			expectedMoves:            []int{},       // e1 blocked by king
			expectedCaptures:         []int{59, 61}, // capture d1 rook, f1 rook
			expectedTriggerPromotion: true,
		},
		{
			name:                     "black pawn promotes on a-file",
			fen:                      "4k3/8/8/8/8/8/p7/4K3 b - - 0 1",
			pieceIndex:               48,        // a2
			expectedMoves:            []int{56}, // a1
			expectedCaptures:         []int{},
			expectedTriggerPromotion: true,
		},
		{
			name:                     "black pawn promotes on h-file",
			fen:                      "4k3/8/8/8/8/8/7p/4K3 b - - 0 1",
			pieceIndex:               55,        // h2
			expectedMoves:            []int{63}, // h1
			expectedCaptures:         []int{},
			expectedTriggerPromotion: true,
		},

		// --- Promotion with check (capturing a piece next to the king) ---
		{
			name:                     "white pawn promotes capturing rook next to black king — gives check",
			fen:                      "3r2k1/4P3/8/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               12,       // e7
			expectedMoves:            []int{4}, // e8
			expectedCaptures:         []int{3}, // capture d8 rook (promotes next to king)
			expectedTriggerPromotion: true,
		},
		{
			name:       "white pawn promotes capturing piece giving discovered check",
			fen:        "1rk5/2P5/8/8/8/8/8/2R1K3 w - - 0 1",
			pieceIndex: 10, // c7
			// capture c8 rook(2) — promotes, and white rook d1(59) gives discovered check on d-file
			// push d8(3) — but that blocks own rook's check... still a promotion
			expectedMoves:            []int{},
			expectedCaptures:         []int{1}, // b8
			expectedTriggerPromotion: true,
		},
		{
			name:                     "black pawn promotes capturing piece next to white king",
			fen:                      "4k3/8/8/8/8/8/4p3/3RK3 b - - 0 1",
			pieceIndex:               52,        // e2
			expectedMoves:            []int{},   // e1 blocked by king
			expectedCaptures:         []int{59}, // capture d1 rook (promotes next to king)
			expectedTriggerPromotion: true,
		},

		// --- NOT a promotion (pawn not on 7th/2nd rank) ---
		{
			name:                     "white pawn on 6th rank does not trigger promotion",
			fen:                      "4k3/8/4P3/8/8/8/8/4K3 w - - 0 1",
			pieceIndex:               20,        // e6
			expectedMoves:            []int{12}, // e7
			expectedCaptures:         []int{},
			expectedTriggerPromotion: false,
		},
		{
			name:                     "black pawn on 3rd rank does not trigger promotion",
			fen:                      "4k3/8/8/8/8/4p3/8/4K3 b - - 0 1",
			pieceIndex:               44,        // e3
			expectedMoves:            []int{52}, // e2
			expectedCaptures:         []int{},
			expectedTriggerPromotion: false,
		},

		// ============================================================
		// CHECKMATE — every piece of the mated side has no legal moves
		// ============================================================

		// --- Back rank mate: R on a8 checks king g8, pawns block escape ---
		// FEN: R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1
		{
			name:                                     "back rank mate — king g8 no moves",
			fen:                                      "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:                               6, // king g8
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "back rank mate — pawn f7 no moves",
			fen:                                      "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:                               13, // f7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "back rank mate — pawn g7 no moves",
			fen:                                      "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:                               14, // g7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "back rank mate — pawn h7 no moves",
			fen:                                      "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:                               15, // h7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// --- Smothered mate: knight f7 checks king h8, own pieces block ---
		// FEN: 6rk/5Npp/8/8/8/8/8/4K3 b - - 0 1

		{
			name:                                     "smothered mate — king h8 no moves",
			fen:                                      "6rk/5Npp/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:                               7, // king h8
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:       "smothered mate — rook g8 no moves",
			fen:        "6rk/5Npp/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex: 6, // rook g8
			// can't block a knight check, can't capture f7 knight
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "smothered mate — pawn g7 no moves",
			fen:                                      "6rk/5Npp/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:                               14, // g7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "smothered mate — pawn h7 no moves",
			fen:                                      "6rk/5Npp/8/8/8/8/8/4K3 b - - 0 1",
			pieceIndex:                               15, // h7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// --- Queen + King mate: queen b7 checks king a8, king c6 covers ---
		// FEN: k7/1Q6/2K5/8/8/8/8/8 b - - 0 1
		{
			name:       "Q+K mate — king a8 no moves",
			fen:        "k7/1Q6/2K5/8/8/8/8/8 b - - 0 1",
			pieceIndex: 0, // king a8
			// b8(1) attacked by queen, a7(8) attacked by queen and king, b7(9) is the queen protected by king
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// --- Rook + King mate on 8th rank: rook b8 checks king g8, king g6 covers ---
		// FEN: 1R4k1/6pp/6K1/8/8/8/8/8 b - - 0 1
		{
			name:       "R+K rank mate — king g8 no moves",
			fen:        "1R4k1/6pp/6K1/8/8/8/8/8 b - - 0 1",
			pieceIndex: 6, // king g8
			// f8(5) attacked by rook, h8(7) attacked by rook
			// f7(13), g7(14), h7(15) all attacked by king g6
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:       "R+K rank mate — pawn g7 no moves",
			fen:        "1R4k1/6pp/6K1/8/8/8/8/8 b - - 0 1",
			pieceIndex: 14, // g7 pawn
			// can't capture rook on b8, can't block rank 8
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "R+K rank mate — pawn h7 no moves",
			fen:                                      "1R4k1/6pp/6K1/8/8/8/8/8 b - - 0 1",
			pieceIndex:                               15, // h7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// White back rank mate
		// FEN: 4k3/8/8/8/8/8/PPP5/1K5r w - - 0 1

		{
			name:                                     "white back rank mate — king b1 no moves",
			fen:                                      "4k3/8/8/8/8/8/PPP5/1K5r w - - 0 1",
			pieceIndex:                               57, // king b1
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "white back rank mate — pawn a2 no moves",
			fen:                                      "4k3/8/8/8/8/8/PPP5/1K5r w - - 0 1",
			pieceIndex:                               48, // a2 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "white back rank mate — pawn b2 no moves",
			fen:                                      "4k3/8/8/8/8/8/PPP5/1K5r w - - 0 1",
			pieceIndex:                               49, // b2 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
		{
			name:                                     "white back rank mate — pawn c2 no moves",
			fen:                                      "4k3/8/8/8/8/8/PPP5/1K5r w - - 0 1",
			pieceIndex:                               50, // c2 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// ============================================================
		// STALEMATE — not in check, but no legal moves
		// ============================================================

		{
			name:                                     "stalemate — king a8 no moves not in check",
			fen:                                      "k7/2Q5/1K6/8/8/8/8/8 b - - 0 1",
			pieceIndex:                               0, // king a8
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: false,
		},

		{
			name:                                     "stalemate — king cornered by rooks",
			fen:                                      "k7/7R/3K4/8/8/8/8/1R6 b - - 1 1",
			pieceIndex:                               0, // king a8
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: false,
		},

		{
			name:                                     "stalemate — king cornered and pawn blocked",
			fen:                                      "k7/2Q5/1K6/6p1/6P1/8/8/8 b - - 0 1",
			pieceIndex:                               30, // pawn g5
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: false,
		},
		{
			name:                                     "stalemate — king cornered and pawn blocked (king query)",
			fen:                                      "k7/2Q5/1K6/6p1/6P1/8/8/8 b - - 0 1",
			pieceIndex:                               0, // king a8
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedFriendlyKingInCheckOrPiecePinned: false,
		},

		// ============================================================
		// WRONG TURN — querying opponent's piece returns nothing
		// ============================================================

		{
			name:             "wrong turn — querying black piece on white's turn",
			fen:              "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:       12, // e7 black pawn
			expectedMoves:    []int{},
			expectedCaptures: []int{},
		},
		{
			name:             "wrong turn — querying white piece on black's turn",
			fen:              "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1",
			pieceIndex:       36, // e4 white pawn
			expectedMoves:    []int{},
			expectedCaptures: []int{},
		},

		// ============================================================
		// EMPTY SQUARE — querying a square with no piece
		// ============================================================

		{
			name:             "empty square returns nothing",
			fen:              "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			pieceIndex:       28, // e5, empty
			expectedMoves:    []int{},
			expectedCaptures: []int{},
		},

		// ============================================================
		// CASTLING — rook under attack should still be legal
		// ============================================================

		{
			name:             "white can queenside castle even if rook a1 is attacked",
			fen:              "4k3/8/8/8/8/8/r1PPPPPP/R3K2R w KQ - 0 1",
			pieceIndex:       60,                    // e1 king
			expectedMoves:    []int{59, 61, 62, 58}, // d1, f1, g1(O-O), c1(O-O-O)
			expectedCaptures: []int{},
		},

		// ============================================================
		// PROMOTION WHILE PINNED
		// ============================================================

		// Pawn on e7 pinned along e-file by rook
		{
			name:                                     "pinned pawn cannot promote",
			fen:                                      "8/3rP1K1/8/8/8/8/8/8 w - - 0 1",
			pieceIndex:                               12, // e7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{}, // capture rook on e8, promotes
			expectedTriggerPromotion:                 true,
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// Pawn on e7 pinned diagonally — cannot promote forward
		{
			name:                                     "diagonally pinned pawn on 7th rank cannot promote forward",
			fen:                                      "4k1K1/5P2/8/8/8/1b6/8/8 w - - 0 1",
			pieceIndex:                               13, // f7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{},
			expectedTriggerPromotion:                 true,
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},

		// Pawn on e7 pinned diagonally — can only promote with capture
		{
			name:                                     "diagonally pinned pawn on 7th rank can only promote diagonally",
			fen:                                      "4k1b1/5P2/8/8/2K5/8/8/8 w - - 0 1",
			pieceIndex:                               13, // f7 pawn
			expectedMoves:                            []int{},
			expectedCaptures:                         []int{6},
			expectedTriggerPromotion:                 true,
			expectedFriendlyKingInCheckOrPiecePinned: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentGameState := BoardFromFEN(tt.fen)
			moves, captures, triggerPromotion, friendlyKingInCheckOrPiecePinned := GetValidMovesForPiece(tt.pieceIndex, currentGameState)

			sort.Ints(moves)
			sort.Ints(tt.expectedMoves)
			if !slices.Equal(moves, tt.expectedMoves) {
				t.Errorf("moves: got %v, want %v", moves, tt.expectedMoves)
			}

			sort.Ints(captures)
			sort.Ints(tt.expectedCaptures)
			if !slices.Equal(captures, tt.expectedCaptures) {
				t.Errorf("captures: got %v, want %v", captures, tt.expectedCaptures)
			}

			if triggerPromotion != tt.expectedTriggerPromotion {
				t.Errorf("triggerPromotion: got %v, want %v", triggerPromotion, tt.expectedTriggerPromotion)
			}

			if friendlyKingInCheckOrPiecePinned != tt.expectedFriendlyKingInCheckOrPiecePinned {
				t.Errorf("friendlyKingInCheck: got %v, want %v", friendlyKingInCheckOrPiecePinned, tt.expectedFriendlyKingInCheckOrPiecePinned)
			}
		})
	}
}
