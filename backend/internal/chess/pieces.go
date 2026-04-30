package chess

import "fmt"

func createPawn(position int, colour pieceColour) pieceType {
	var moves, attacks []int
	if colour == White {
		moves = []int{-8}
		attacks = []int{-7, -9}
	} else {
		moves = []int{8}
		attacks = []int{7, 9}
	}
	return pieceType{
		position:           position,
		colour:             colour,
		variant:            Pawn,
		moves:              moves,
		attacks:            attacks,
		moveRange:          1,
		attackRange:        1,
		movesEqualsAttacks: false,
	}
}

func createKnight(position int, colour pieceColour) pieceType {
	return pieceType{
		position:           position,
		colour:             colour,
		variant:            Knight,
		moves:              []int{-15, -6, 10, 17, 15, 6, -10, -17},
		moveRange:          1,
		movesEqualsAttacks: true,
	}
}

func createBishop(position int, colour pieceColour) pieceType {
	return pieceType{
		position:           position,
		colour:             colour,
		variant:            Bishop,
		moves:              []int{-9, -7, 7, 9},
		moveRange:          7,
		movesEqualsAttacks: true,
	}
}

func createRook(position int, colour pieceColour) pieceType {
	return pieceType{
		position:           position,
		colour:             colour,
		variant:            Rook,
		moves:              []int{-8, 8, -1, 1},
		moveRange:          7,
		movesEqualsAttacks: true,
	}
}

func createQueen(position int, colour pieceColour) pieceType {
	return pieceType{
		position:           position,
		colour:             colour,
		variant:            Queen,
		moves:              []int{-9, -8, -7, 1, 9, 8, 7, -1},
		moveRange:          7,
		movesEqualsAttacks: true,
	}
}

func createKing(position int, colour pieceColour) pieceType {
	return pieceType{
		position:           position,
		colour:             colour,
		variant:            King,
		moves:              []int{-9, -8, -7, 1, 9, 8, 7, -1},
		moveRange:          1,
		movesEqualsAttacks: true,
	}
}

func createPiece(position int, colour pieceColour, variant pieceVariant) *pieceType {
	var newPiece pieceType
	switch variant {
	case Pawn:
		newPiece = createPawn(position, colour)
	case Knight:
		newPiece = createKnight(position, colour)
	case Bishop:
		newPiece = createBishop(position, colour)
	case Rook:
		newPiece = createRook(position, colour)
	case Queen:
		newPiece = createQueen(position, colour)
	case King:
		newPiece = createKing(position, colour)
	default:
		fmt.Println("Unknown piece variant")
	}
	return &newPiece
}
