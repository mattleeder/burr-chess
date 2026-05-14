package chess

import (
	"log"
	"math"
	"os"
)

var errorLog = log.New(os.Stderr, "CHESS ERROR\t", log.Ldate|log.Ltime|log.Llongfile)

type pieceColour int

const (
	White = iota
	Black
)

type pieceVariant int

const (
	Pawn pieceVariant = iota
	Knight
	Bishop
	Rook
	Queen
	King
)

type GameOverStatusCode int

const (
	Ongoing GameOverStatusCode = iota
	Stalemate
	Checkmate
	ThreefoldRepetition
	InsufficientMaterial
	WhiteFlagged
	BlackFlagged
	Draw
	WhiteResigned
	BlackResigned
	WhiteDisconnected
	BlackDisconnected
	GameAborted
)

type MatchOutcome int

const (
	OutcomeDraw      MatchOutcome = iota // 0
	OutcomeWhiteWins                     // 1
	OutcomeBlackWins                     // 2
)

func (o MatchOutcome) String() string {
	switch o {
	case OutcomeDraw:
		return "draw"
	case OutcomeWhiteWins:
		return "white_wins"
	case OutcomeBlackWins:
		return "black_wins"
	default:
		return "unknown"
	}
}

func (c GameOverStatusCode) String() string {
	switch c {
	case Ongoing:
		return "ongoing"
	case Stalemate:
		return "stalemate"
	case Checkmate:
		return "checkmate"
	case ThreefoldRepetition:
		return "threefold_repetition"
	case InsufficientMaterial:
		return "insufficient_material"
	case WhiteFlagged:
		return "white_flagged"
	case BlackFlagged:
		return "black_flagged"
	case Draw:
		return "draw"
	case WhiteResigned:
		return "white_resigned"
	case BlackResigned:
		return "black_resigned"
	case WhiteDisconnected:
		return "white_disconnected"
	case BlackDisconnected:
		return "black_disconnected"
	case GameAborted:
		return "game_aborted"
	default:
		return "unknown"
	}
}

var (
	Bullet    = [2]int64{0, 2 * 60_000}
	Blitz     = [2]int64{Bullet[1], 5 * 60_000}
	Rapid     = [2]int64{Blitz[1], 20 * 60_000}
	Classical = [2]int64{Rapid[1], math.MaxInt64}
)

type pieceType struct {
	position               int
	colour                 pieceColour
	variant                pieceVariant
	moves, attacks         []int
	moveRange, attackRange int
	movesEqualsAttacks     bool
}

type square struct {
	piece          *pieceType
	whiteAttacking bool
	blackAttacking bool
}

// Board layout: index 0 = a8 (top-left), index 63 = h1 (bottom-right).
// White pieces start on indices 48-63 (ranks 1-2), Black on 0-15 (ranks 7-8).
// White pawns promote on indices 0-7 (rank 8), Black on 56-63 (rank 1).
type gameState struct {
	board                   [64]square
	turn                    pieceColour
	blackKingPosition       int
	whiteKingPosition       int
	blackCanKingSideCastle  bool
	blackCanQueenSideCastle bool
	whiteCanKingSideCastle  bool
	whiteCanQueenSideCastle bool
	enPassantTargetSquare   int
	enPassantAvailable      bool
	halfMoveClock           int
	fullMoveNumber          int
}

// Package-level lookup maps

var runeToVariant = map[rune]pieceVariant{
	'p': Pawn, 'n': Knight, 'b': Bishop, 'r': Rook, 'q': Queen, 'k': King,
	'P': Pawn, 'N': Knight, 'B': Bishop, 'R': Rook, 'Q': Queen, 'K': King,
}

var stringToVariant = map[string]pieceVariant{
	"n": Knight, "b": Bishop, "r": Rook, "q": Queen,
}

var variantToString = map[pieceVariant]string{
	Knight: "n", Bishop: "b", Rook: "r", Queen: "q",
}

var variantToRune = map[pieceVariant]rune{
	Pawn: 'p', Knight: 'n', Bishop: 'b', Rook: 'r', Queen: 'q', King: 'k',
}

var fileToInt = map[rune]int{
	'a': 0, 'b': 1, 'c': 2, 'd': 3, 'e': 4, 'f': 5, 'g': 6, 'h': 7,
}

var intToFile = map[int]rune{
	0: 'a', 1: 'b', 2: 'c', 3: 'd', 4: 'e', 5: 'f', 6: 'g', 7: 'h',
}

var intToRune = map[int]rune{
	1: '1', 2: '2', 3: '3', 4: '4', 5: '5', 6: '6', 7: '7', 8: '8',
}

// Utility functions

func filter[T any](arr []T, fn func(T) bool) []T {
	result := []T{}
	for _, v := range arr {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

func lambdaMapGet[T comparable, U any](m map[T]U) func(T) U {
	return func(key T) U {
		return m[key]
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func getRow(position int) int {
	return position / 8
}

func getCol(position int) int {
	return position % 8
}

func defaultSquare() square {
	return square{nil, false, false}
}

func isSquareInBoard(square int) bool {
	return square >= 0 && square < 64
}
