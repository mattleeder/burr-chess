package chess

import (
	"fmt"
	"strconv"
	"unicode"
)

func intToAlgebraicNotation(position int) string {
	var columns = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	row := 8 - position/8
	col := position % 8
	return fmt.Sprintf("%s%v", columns[col], row)
}

func BoardFromFEN(fen string) gameState {
	var colour pieceColour
	var variant pieceVariant
	var boardIndex = 0
	var board [64]square
	var parseState = 0
	var currentGameState gameState

	for _, char := range fen {
		if char == ' ' {
			parseState++
			continue
		}

		switch parseState {
		case 0:
			// Parse Board
			if char == '/' {
				continue
			}
			if unicode.IsDigit(char) {
				for i := 0; i < int(char-'0'); i++ {
					board[boardIndex] = defaultSquare()
					boardIndex++
				}
				continue
			}
			if unicode.IsUpper(char) {
				colour = White
			} else {
				colour = Black
			}
			variant = runeToVariant[char]
			board[boardIndex] = defaultSquare()
			board[boardIndex].piece = createPiece(boardIndex, colour, variant)
			if variant == King {
				if colour == White {
					currentGameState.whiteKingPosition = boardIndex
				} else {
					currentGameState.blackKingPosition = boardIndex
				}
			}
			boardIndex++

		case 1:
			currentGameState.board = board
			// Parse Turn
			if char == 'w' {
				currentGameState.turn = White
			} else {
				currentGameState.turn = Black
			}

		case 2:
			// Parse Castling
			switch char {
			case 'K':
				currentGameState.whiteCanKingSideCastle = true
			case 'Q':
				currentGameState.whiteCanQueenSideCastle = true
			case 'k':
				currentGameState.blackCanKingSideCastle = true
			case 'q':
				currentGameState.blackCanQueenSideCastle = true
			}

		case 3:
			// Parse En passant
			if char == '-' {
				continue
			}
			currentGameState.enPassantAvailable = true
			if unicode.IsLetter(char) {
				currentGameState.enPassantTargetSquare += fileToInt[char]
			}
			if unicode.IsDigit(char) {
				currentGameState.enPassantTargetSquare += (8 - int(char-'0')) * 8
			}

		case 4:
			// Parse Halfmove Clock
			currentGameState.halfMoveClock *= 10
			val, err := strconv.Atoi(string(char))
			if err != nil {
				errorLog.Println(err)
			}
			currentGameState.halfMoveClock += val

		case 5:
			// Parse fullmove number
			currentGameState.fullMoveNumber *= 10
			val, err := strconv.Atoi(string(char))
			if err != nil {
				errorLog.Println(err)
			}
			currentGameState.fullMoveNumber += val
		}
	}

	return currentGameState
}

func gameStateToFEN(newGameState gameState) string {
	var newFEN []rune
	var rowCount int
	var emptyCount int
	var colour pieceColour
	var variant pieceVariant
	var char rune

	for _, value := range newGameState.board {
		rowCount++

		if value.piece == nil {
			emptyCount++
		} else {
			colour = value.piece.colour
			variant = value.piece.variant

			if emptyCount > 0 {
				newFEN = append(newFEN, intToRune[emptyCount])
				emptyCount = 0
			}

			char = variantToRune[variant]
			if colour == White {
				char = unicode.To(0, char)
			}
			newFEN = append(newFEN, char)
		}

		if rowCount >= 8 {
			if emptyCount > 0 {
				newFEN = append(newFEN, intToRune[emptyCount])
				emptyCount = 0
			}
			newFEN = append(newFEN, '/')
			rowCount = 0
		}
	}

	newFEN = newFEN[:len(newFEN)-1]
	newFEN = append(newFEN, ' ')

	// Turn
	if newGameState.turn == White {
		newFEN = append(newFEN, 'w')
	} else {
		newFEN = append(newFEN, 'b')
	}
	newFEN = append(newFEN, ' ')

	// Castling
	if newGameState.whiteCanKingSideCastle {
		newFEN = append(newFEN, 'K')
	}
	if newGameState.whiteCanQueenSideCastle {
		newFEN = append(newFEN, 'Q')
	}
	if newGameState.blackCanKingSideCastle {
		newFEN = append(newFEN, 'k')
	}
	if newGameState.blackCanQueenSideCastle {
		newFEN = append(newFEN, 'q')
	}
	if newFEN[len(newFEN)-1] == ' ' {
		newFEN = append(newFEN, '-')
	}
	newFEN = append(newFEN, ' ')

	// En passant
	if newGameState.enPassantAvailable {
		file := newGameState.enPassantTargetSquare % 8
		rank := 8 - (newGameState.enPassantTargetSquare / 8)
		newFEN = append(newFEN, intToFile[file])
		newFEN = append(newFEN, intToRune[rank])
	} else {
		newFEN = append(newFEN, '-')
	}
	newFEN = append(newFEN, ' ')

	newFEN = append(newFEN, []rune(fmt.Sprint(newGameState.halfMoveClock))...)
	newFEN = append(newFEN, ' ')
	newFEN = append(newFEN, []rune(fmt.Sprint(newGameState.fullMoveNumber))...)

	return string(newFEN)
}
