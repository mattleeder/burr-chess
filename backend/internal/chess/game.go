package chess

import (
	"slices"
)

// buildAlgebraicNotation constructs the algebraic notation prefix for a move
// (piece identifier, disambiguation, capture indicator, and destination square).
func buildAlgebraicNotation(piece int, move int, currentGameState gameState) string {
	var algebraicNotation = ""
	var pawnFile = ""
	var oldPositionAlgebraic = intToAlgebraicNotation(piece)

	if currentGameState.board[piece].piece.variant == Pawn {
		pawnFile = string(oldPositionAlgebraic[0])
	} else {
		algebraicNotation += variantToString[currentGameState.board[piece].piece.variant]
	}

	// Disambiguation: can another piece of the same type reach the same square?
	var currentPieceColour = currentGameState.board[piece].piece.colour
	var currentPieceVariant = currentGameState.board[piece].piece.variant
	var rankDistinguishNeeded = false
	var fileDistinguishNeeded = false

	for i := 0; i < 64; i++ {
		if i == piece {
			continue
		}
		if currentGameState.board[i].piece == nil || currentGameState.board[i].piece.colour != currentPieceColour || currentGameState.board[i].piece.variant != currentPieceVariant {
			continue
		}
		moves, captures, _, _ := getMovesandCapturesForPiece(i, currentGameState)
		if slices.Contains(append(moves, captures...), move) {
			if (i-piece)%8 == 0 {
				rankDistinguishNeeded = true
			} else {
				fileDistinguishNeeded = true
			}
		}
	}

	if fileDistinguishNeeded {
		algebraicNotation += string(oldPositionAlgebraic[0])
	}
	if rankDistinguishNeeded {
		algebraicNotation += string(oldPositionAlgebraic[1])
	}

	// Capture detection (piece present or pawn diagonal = en passant)
	if currentGameState.board[move].piece != nil || (pawnFile != "" && (move-piece)%8 != 0) {
		algebraicNotation = pawnFile + algebraicNotation + "x"
	}
	algebraicNotation += intToAlgebraicNotation(move)

	return algebraicNotation
}

// applyMove mutates the game state by applying a move, handling castling, en passant,
// promotion, and castling rights updates. Returns updated gameState and any notation suffix.
func applyMove(currentGameState gameState, piece int, move int, promotionString string) (gameState, string) {
	currentGameState.board[move].piece = currentGameState.board[piece].piece
	currentGameState.board[piece].piece = nil

	var newGameState = currentGameState
	newGameState.halfMoveClock += 1
	if newGameState.halfMoveClock%2 == 0 {
		newGameState.fullMoveNumber += 1
	}

	var notationSuffix = ""

	// Check for promotion
	if newGameState.board[move].piece.variant == Pawn && (move <= 7 || move >= 56) {
		var promotionColour pieceColour = Black
		if move <= 7 {
			promotionColour = White
		}
		promotionVariant, ok := stringToVariant[promotionString]
		if !ok {
			errorLog.Println("Could not understand promotion string")
		}
		newGameState.board[move].piece = createPiece(move, promotionColour, promotionVariant)
		notationSuffix = "=" + promotionString
	}

	// Check for king move
	if newGameState.board[move].piece.variant == King {
		if abs(move-piece) == 2 {
			if newGameState.board[move+1].piece != nil && newGameState.board[move+1].piece.variant == Rook {
				newGameState.board[move-1].piece = newGameState.board[move+1].piece
				newGameState.board[move+1].piece = nil
				notationSuffix = "::O-O" // signal to replace full notation
			} else if newGameState.board[move-2].piece != nil && newGameState.board[move-2].piece.variant == Rook {
				newGameState.board[move+1].piece = newGameState.board[move-2].piece
				newGameState.board[move-2].piece = nil
				notationSuffix = "::O-O-O" // signal to replace full notation
			}
		}
		if newGameState.turn == White {
			newGameState.whiteCanKingSideCastle = false
			newGameState.whiteCanQueenSideCastle = false
		} else {
			newGameState.blackCanKingSideCastle = false
			newGameState.blackCanQueenSideCastle = false
		}
	}

	// Update castling rights for rook moves/captures
	if move == 0 || piece == 0 {
		newGameState.blackCanQueenSideCastle = false
	}
	if move == 7 || piece == 7 {
		newGameState.blackCanKingSideCastle = false
	}
	if move == 56 || piece == 56 {
		newGameState.whiteCanQueenSideCastle = false
	}
	if move == 63 || piece == 63 {
		newGameState.whiteCanKingSideCastle = false
	}

	// En passant capture
	if move == newGameState.enPassantTargetSquare && newGameState.board[move].piece.variant == Pawn && abs(move-piece) != 8 {
		if newGameState.board[move].piece.colour == White {
			newGameState.board[move+8].piece = nil
		} else {
			newGameState.board[move-8].piece = nil
		}
	}

	// En passant availability
	if newGameState.board[move].piece.variant == Pawn && abs(move-piece) == 16 {
		newGameState.enPassantAvailable = true
		if newGameState.turn == White {
			newGameState.enPassantTargetSquare = move + 8
		} else {
			newGameState.enPassantTargetSquare = move - 8
		}
	} else {
		newGameState.enPassantAvailable = false
	}

	// Change Turn
	if newGameState.turn == White {
		newGameState.turn = Black
	} else {
		newGameState.turn = White
	}

	return newGameState, notationSuffix
}

func gameHasSufficientMaterial(currentGameState gameState) bool {
	var piece *pieceType
	var freqDict = make(map[pieceVariant]int)
	var totalPieceCount int
	var darkSquareBishopPresent = false
	var lightSquareBishopPresent = false
	for i := range currentGameState.board {
		piece = currentGameState.board[i].piece
		if piece != nil {
			freqDict[piece.variant] += 1
			totalPieceCount += 1
			if piece.variant == Bishop {
				row := getRow(i)
				col := getCol(i)
				if (row+col)%2 == 0 {
					lightSquareBishopPresent = true
				} else {
					darkSquareBishopPresent = true
				}
			}
		}
	}

	if totalPieceCount == 2 {
		return false
	}
	if totalPieceCount == 3 && (freqDict[Knight] >= 1 || freqDict[Bishop] >= 1) {
		return false
	}
	if totalPieceCount-2 == freqDict[Bishop] && !(darkSquareBishopPresent && lightSquareBishopPresent) {
		return false
	}
	return true
}

func GetFENAfterMove(currentFEN string, piece int, move int, promotionString string) (string, GameOverStatusCode, string) {
	var currentGameState = BoardFromFEN(currentFEN)

	// Build algebraic notation
	algebraicNotation := buildAlgebraicNotation(piece, move, currentGameState)

	// Apply the move
	newGameState, notationSuffix := applyMove(currentGameState, piece, move, promotionString)

	// Handle castling notation (replaces the full notation)
	if len(notationSuffix) > 2 && notationSuffix[:2] == "::" {
		algebraicNotation = notationSuffix[2:]
	} else {
		algebraicNotation += notationSuffix
	}

	newFEN := gameStateToFEN(newGameState)

	// BUG FIX: was checking currentGameState (pre-move), now checks newGameState (post-move)
	if !gameHasSufficientMaterial(newGameState) {
		return newFEN, InsufficientMaterial, algebraicNotation
	}

	// Check for checkmate / stalemate
	var gameOverStatus GameOverStatusCode = Ongoing
	var enemyKing = newGameState.blackKingPosition
	var enemyKingColour = Black
	if currentGameState.board[piece].piece.colour == Black {
		enemyKing = newGameState.whiteKingPosition
		enemyKingColour = White
	}

	var moves, captures, _, enemyKingInCheck = GetValidMovesForPiece(enemyKing, newGameState)
	var enemyKingMoves = append(moves, captures...)

	if len(enemyKingMoves) == 0 {
		if !canColourMove(newGameState, pieceColour(enemyKingColour)) {
			if enemyKingInCheck {
				gameOverStatus = Checkmate
			} else {
				gameOverStatus = Stalemate
			}
		}
	}

	if gameOverStatus == Checkmate {
		algebraicNotation += "#"
	} else if enemyKingInCheck {
		algebraicNotation += "+"
	}

	return newFEN, gameOverStatus, algebraicNotation
}
