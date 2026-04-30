package chess

func hasQueenCrossedEdgeThroughDiagonal(direction int, piecePosition int, movePosition int) bool {
	if !(abs(direction) == 7 || abs(direction) == 9) {
		return false
	}

	var pieceRow = getRow(piecePosition)
	var pieceCol = getCol(piecePosition)
	var moveRow = getRow(movePosition)
	var moveCol = getCol(movePosition)
	var rowChange = abs(pieceRow - moveRow)
	var colChange = abs(pieceCol - moveCol)

	return rowChange != colChange
}

func hasMoveCrossedEdge(piecePosition int, movePosition int, pieceVariant pieceVariant) bool {
	var pieceRow = getRow(piecePosition)
	var pieceCol = getCol(piecePosition)
	var moveRow = getRow(movePosition)
	var moveCol = getCol(movePosition)
	var rowChange = abs(pieceRow - moveRow)
	var colChange = abs(pieceCol - moveCol)

	switch pieceVariant {
	case Pawn:
		if colChange > 1 {
			return true
		}
	case Knight:
		if rowChange+colChange != 3 {
			return true
		}
	case Bishop:
		if rowChange != colChange {
			return true
		}
	case Rook:
		if rowChange > 0 && colChange > 0 {
			return true
		}
	case Queen:
		if (rowChange != colChange) && (rowChange > 0 && colChange > 0) {
			return true
		}
	case King:
		if rowChange > 1 || colChange > 1 {
			return true
		}
	}

	return false
}

func isSquareUnderAttack(board [64]square, position int, defendingColour pieceColour) bool {
	var attackingSquare int
	var targetPiece *pieceType

	// Check Pawns
	var possibleAttackingPawnPositions [2]int
	if defendingColour == White {
		possibleAttackingPawnPositions = [2]int{-7, -9}
	} else {
		possibleAttackingPawnPositions = [2]int{7, 9}
	}
	for i := range possibleAttackingPawnPositions {
		attackingSquare = position + possibleAttackingPawnPositions[i]
		if !isSquareInBoard(attackingSquare) {
			continue
		}
		if hasMoveCrossedEdge(position, attackingSquare, Pawn) {
			continue
		}
		targetPiece = board[attackingSquare].piece
		if targetPiece != nil && targetPiece.colour != defendingColour && targetPiece.variant == Pawn {
			return true
		}
	}

	// Check Knights
	var dummyKnight = createKnight(position, defendingColour)
	for _, attackDirection := range dummyKnight.moves {
		attackingSquare = position + attackDirection
		if !isSquareInBoard(attackingSquare) {
			continue
		}
		if hasMoveCrossedEdge(position, attackingSquare, Knight) {
			continue
		}
		targetPiece = board[attackingSquare].piece
		if targetPiece != nil && targetPiece.variant == Knight && targetPiece.colour != defendingColour {
			return true
		}
	}

	// Check Rooks/Queens (straight lines)
	var dummyRook = createRook(position, defendingColour)
	for _, attackDirection := range dummyRook.moves {
		for attackRange := 1; attackRange <= dummyRook.moveRange; attackRange++ {
			attackingSquare = position + (attackDirection * attackRange)
			if !isSquareInBoard(attackingSquare) {
				break
			}
			if hasMoveCrossedEdge(position, attackingSquare, Rook) {
				break
			}
			targetPiece = board[attackingSquare].piece
			if targetPiece == nil || (targetPiece.variant == King && targetPiece.colour == defendingColour) {
				continue
			}
			if (targetPiece.variant == Rook || targetPiece.variant == Queen || (targetPiece.variant == King && attackRange == 1)) && targetPiece.colour != defendingColour {
				return true
			}
			break
		}
	}

	// Check Bishops/Queens (diagonals)
	var dummyBishop = createBishop(position, defendingColour)
	for _, attackDirection := range dummyBishop.moves {
		for attackRange := 1; attackRange <= dummyBishop.moveRange; attackRange++ {
			attackingSquare = position + (attackDirection * attackRange)
			if !isSquareInBoard(attackingSquare) {
				continue
			}
			if hasMoveCrossedEdge(position, attackingSquare, Bishop) {
				break
			}
			targetPiece = board[attackingSquare].piece
			if targetPiece == nil || (targetPiece.variant == King && targetPiece.colour == defendingColour) {
				continue
			}
			if (targetPiece.variant == Bishop || targetPiece.variant == Queen || (targetPiece.variant == King && attackRange == 1)) && targetPiece.colour != defendingColour {
				return true
			}
			break
		}
	}

	return false
}

// findChecksAndPins detects checks and pins against the friendly king.
// Returns the set of squares that can block/capture to resolve check, and the number of checks.
func findChecksAndPins(piece *pieceType, friendlyKingPosition int, board [64]square, enPassantActive bool, enpassantSquare int) (blockingSquares map[int]bool, checkCount int) {
	blockingSquares = make(map[int]bool)
	var currentSquare int
	var targetPiece *pieceType
	var intermediateBlockingSquares []int

	// Check Pawns
	var possibleAttackingPawnPositions [2]int
	if piece.colour == White {
		possibleAttackingPawnPositions = [2]int{-7, -9}
	} else {
		possibleAttackingPawnPositions = [2]int{7, 9}
	}
	for i := range possibleAttackingPawnPositions {
		currentSquare = friendlyKingPosition + possibleAttackingPawnPositions[i]
		if !isSquareInBoard(currentSquare) {
			continue
		}
		if hasMoveCrossedEdge(friendlyKingPosition, currentSquare, Pawn) {
			continue
		}
		targetPiece = board[currentSquare].piece
		if targetPiece != nil && targetPiece.colour != piece.colour && targetPiece.variant == Pawn {
			blockingSquares[currentSquare] = true
			if enPassantActive {
				blockingSquares[enpassantSquare] = true
			}
			checkCount++
		}
	}

	// Check Knights
	dummyKnight := createKnight(friendlyKingPosition, White)
	for i := range dummyKnight.moves {
		currentSquare = dummyKnight.moves[i] + friendlyKingPosition
		if !isSquareInBoard(currentSquare) {
			continue
		}
		if hasMoveCrossedEdge(friendlyKingPosition, currentSquare, Knight) {
			continue
		}
		targetPiece = board[currentSquare].piece
		if targetPiece != nil && targetPiece.colour != piece.colour && targetPiece.variant == Knight {
			blockingSquares[currentSquare] = true
			checkCount++
		}
	}

	// Check horizontal and vertical (Rook/Queen)
	dummyRook := createRook(friendlyKingPosition, White)
	for i := range dummyRook.moves {
		moveDirection := dummyRook.moves[i]
		intermediateBlockingSquares = []int{}

		for moveRange := 1; moveRange <= dummyRook.moveRange; moveRange++ {
			currentSquare = friendlyKingPosition + (moveDirection * moveRange)
			if !isSquareInBoard(currentSquare) {
				break
			}
			if hasMoveCrossedEdge(friendlyKingPosition, currentSquare, Rook) {
				continue
			}
			targetPiece = board[currentSquare].piece
			if targetPiece == nil {
				intermediateBlockingSquares = append(intermediateBlockingSquares, currentSquare)
				continue
			}
			if targetPiece == piece {
				continue
			}
			if (targetPiece.variant == Rook || targetPiece.variant == Queen || (targetPiece.variant == King && moveRange == 1)) && targetPiece.colour != piece.colour {
				intermediateBlockingSquares = append(intermediateBlockingSquares, currentSquare)
				for _, v := range intermediateBlockingSquares {
					blockingSquares[v] = true
				}
				checkCount++
			}
			break
		}
	}

	// Check diagonal (Bishop/Queen)
	dummyBishop := createBishop(friendlyKingPosition, White)
	for i := range dummyBishop.moves {
		moveDirection := dummyBishop.moves[i]
		intermediateBlockingSquares = []int{}

		for moveRange := 1; moveRange <= dummyBishop.moveRange; moveRange++ {
			currentSquare = friendlyKingPosition + (moveDirection * moveRange)
			if !isSquareInBoard(currentSquare) {
				break
			}
			if hasMoveCrossedEdge(friendlyKingPosition, currentSquare, Bishop) {
				continue
			}
			targetPiece = board[currentSquare].piece
			if targetPiece == nil {
				intermediateBlockingSquares = append(intermediateBlockingSquares, currentSquare)
				continue
			}
			if targetPiece == piece {
				continue
			}
			if (targetPiece.variant == Bishop || targetPiece.variant == Queen || (targetPiece.variant == King && moveRange == 1)) && targetPiece.colour != piece.colour {
				intermediateBlockingSquares = append(intermediateBlockingSquares, currentSquare)
				for _, v := range intermediateBlockingSquares {
					blockingSquares[v] = true
				}
				checkCount++
			}
			break
		}
	}

	return blockingSquares, checkCount
}

// getPawnSpecialMoves handles double-move, en passant, and promotion detection.
func getPawnSpecialMoves(piece *pieceType, piecePosition int, board [64]square, enPassantActive bool, enpassantSquare int) (moves []int, captures []int, triggerPromotion bool) {
	// Promotion detection
	if (8 <= piecePosition && piecePosition <= 15 && piece.colour == White) || (48 <= piecePosition && piecePosition <= 55 && piece.colour == Black) {
		triggerPromotion = true
	}

	// Double move
	if (piece.colour == Black && piece.position/8 == 1 && board[piece.position+8].piece == nil) || (piece.colour == White && piece.position/8 == 6 && board[piece.position-8].piece == nil) {
		currentSquare := piece.position + (piece.moves[0] * 2)
		if board[currentSquare].piece == nil {
			moves = append(moves, currentSquare)
		}
	}

	// En passant
	if enPassantActive && !enPassantWouldCauseCheckAlongHorizontal(piece, board, enpassantSquare) {
		if piece.colour == Black && (piece.position+7 == enpassantSquare || piece.position+9 == enpassantSquare) {
			if !hasMoveCrossedEdge(piecePosition, enpassantSquare, Pawn) {
				captures = append(captures, enpassantSquare)
			}
		}
		if piece.colour == White && (piece.position-7 == enpassantSquare || piece.position-9 == enpassantSquare) {
			if !hasMoveCrossedEdge(piecePosition, enpassantSquare, Pawn) {
				captures = append(captures, enpassantSquare)
			}
		}
	}

	return moves, captures, triggerPromotion
}

// enPassantWouldCauseCheckAlongHorizontal handles the edge case where an enemy rook or queen would attack the king from a horizontal after an en passant capture
func enPassantWouldCauseCheckAlongHorizontal(piece *pieceType, board [64]square, enPassantPosition int) bool {
	// Identify horizontal direction then iterate, if we find an enemy rook then return true otherwise return false

	// Capture position defaults to towards white
	var capturePosition int

	if piece.colour == White {
		capturePosition = enPassantPosition + 8
	} else {
		capturePosition = enPassantPosition - 8
	}

	row := getRow(piece.position)
	rowStart := row * 8
	rowEnd := rowStart + 7

	// Iterate
	var currentPosition = rowStart

	var lastPiece *pieceType = nil
	var newPiece *pieceType = nil

	// Iterate to identify if horizontal line between friendly king and enemy rook or queen
	for currentPosition <= rowEnd {

		// Empty square, continue iterating
		if board[currentPosition].piece == nil {
			currentPosition += 1
			continue
		}

		// Skip enpassant pieces
		if currentPosition == piece.position || currentPosition == capturePosition {
			currentPosition += 1
			continue
		}

		// Piece
		newPiece = board[currentPosition].piece

		// Next two are a little bit gross

		// New piece is friendly king and last piece is enemy rook or queen
		if newPiece.colour == piece.colour && newPiece.variant == King {
			if lastPiece != nil && lastPiece.colour != piece.colour && (lastPiece.variant == Queen || lastPiece.variant == Rook) {
				return true
			}
		}

		// New piece is enemy rook or queen and last piece is friendly king
		if newPiece.colour != piece.colour && (newPiece.variant == Queen || newPiece.variant == Rook) {
			if lastPiece != nil && lastPiece.colour == piece.colour && lastPiece.variant == King {
				return true
			}
		}

		lastPiece = newPiece
		currentPosition += 1
	}

	// Edge of board
	return false
}

// getKingMoves generates king moves including castling.
func getKingMoves(piece *pieceType, piecePosition int, board [64]square, shortCastleAvailable bool, longCastleAvailable bool, checkCount int) (moves []int, captures []int) {
	for _, moveDirection := range piece.moves {
		currentSquare := piece.position + moveDirection
		if !isSquareInBoard(currentSquare) {
			continue
		}
		if hasMoveCrossedEdge(piecePosition, currentSquare, King) {
			continue
		}
		targetPiece := board[currentSquare].piece
		if targetPiece == nil || targetPiece.colour != piece.colour {
			if isSquareUnderAttack(board, currentSquare, piece.colour) {
				continue
			}
			if targetPiece != nil {
				captures = append(captures, currentSquare)
			} else {
				moves = append(moves, currentSquare)
			}
		}
	}

	if shortCastleAvailable && checkCount == 0 {
		if board[piece.position+1].piece == nil && board[piece.position+2].piece == nil {
			if !isSquareUnderAttack(board, piece.position+1, piece.colour) && !isSquareUnderAttack(board, piece.position+2, piece.colour) {
				moves = append(moves, piece.position+2)
			}
		}
	}

	if longCastleAvailable && checkCount == 0 {
		if board[piece.position-1].piece == nil && board[piece.position-2].piece == nil && board[piece.position-3].piece == nil {
			if !isSquareUnderAttack(board, piece.position-1, piece.colour) && !isSquareUnderAttack(board, piece.position-2, piece.colour) && !isSquareUnderAttack(board, piece.position-3, piece.colour) {
				moves = append(moves, piece.position-2)
			}
		}
	}

	return moves, captures
}

func getMovesandCapturesForPiece(piecePosition int, currentGameState gameState) (moves []int, captures []int, triggerPromotion bool, friendlyKingInCheckOrPiecePinned bool) {
	var currentSquare int
	var targetPiece *pieceType
	var moveDirection int
	var board = currentGameState.board
	var piece = board[piecePosition].piece
	triggerPromotion = false

	if piece == nil {
		return []int{}, []int{}, false, false
	}

	if piece.colour != currentGameState.turn {
		return []int{}, []int{}, false, false
	}

	var shortCastleAvailable bool
	var longCastleAvailable bool
	var friendlyKingPosition int

	if piece.colour == White {
		shortCastleAvailable = currentGameState.whiteCanKingSideCastle
		longCastleAvailable = currentGameState.whiteCanQueenSideCastle
		friendlyKingPosition = currentGameState.whiteKingPosition
	} else {
		shortCastleAvailable = currentGameState.blackCanKingSideCastle
		longCastleAvailable = currentGameState.blackCanQueenSideCastle
		friendlyKingPosition = currentGameState.blackKingPosition
	}

	// Basic moves for non-kings
	if piece.variant != King {
		for i := range piece.moves {
			moveDirection = piece.moves[i]
			for moveRange := 1; moveRange <= piece.moveRange; moveRange++ {
				currentSquare = piece.position + (moveDirection * moveRange)
				if !isSquareInBoard(currentSquare) {
					break
				}
				if hasMoveCrossedEdge(piecePosition, currentSquare, piece.variant) {
					break
				}
				if piece.variant == Queen && hasQueenCrossedEdgeThroughDiagonal(moveDirection, piecePosition, currentSquare) {
					break
				}
				targetPiece = board[currentSquare].piece
				if targetPiece != nil {
					if !piece.movesEqualsAttacks || targetPiece.colour == piece.colour || targetPiece.variant == King {
						break
					}
					captures = append(captures, currentSquare)
					break
				}
				moves = append(moves, currentSquare)
			}
		}
	}

	// Basic attacks (for pieces where moves != attacks, i.e. pawns)
	if !piece.movesEqualsAttacks {
		for i := range piece.attacks {
			attackDirection := piece.attacks[i]
			for attackRange := 1; attackRange <= piece.moveRange; attackRange++ {
				currentSquare = piece.position + (attackDirection * attackRange)
				if !isSquareInBoard(currentSquare) {
					break
				}
				if hasMoveCrossedEdge(piecePosition, currentSquare, piece.variant) {
					break
				}
				targetPiece = board[currentSquare].piece
				if targetPiece == nil {
					continue
				}
				if targetPiece.colour == piece.colour || targetPiece.variant == King {
					break
				}
				captures = append(captures, currentSquare)
				break
			}
		}
	}

	// Pawn special moves
	if piece.variant == Pawn {
		pawnMoves, pawnCaptures, pawnPromotion := getPawnSpecialMoves(piece, piecePosition, board, currentGameState.enPassantAvailable, currentGameState.enPassantTargetSquare)
		moves = append(moves, pawnMoves...)
		captures = append(captures, pawnCaptures...)
		if pawnPromotion {
			triggerPromotion = true
		}
	}

	// Check/pin detection
	blockingSquares, checkCount := findChecksAndPins(piece, friendlyKingPosition, board, currentGameState.enPassantAvailable, currentGameState.enPassantTargetSquare)

	// King moves
	if piece.variant == King {
		kingMoves, kingCaptures := getKingMoves(piece, piecePosition, board, shortCastleAvailable, longCastleAvailable, checkCount)
		moves = append(moves, kingMoves...)
		captures = append(captures, kingCaptures...)
	}

	// Double check: king must move
	if checkCount >= 2 {
		return []int{}, []int{}, false, true
	}

	// Single check: must block or capture
	if checkCount == 1 && piece.variant != King {
		moves = filter(moves, lambdaMapGet(blockingSquares))
		captures = filter(captures, lambdaMapGet(blockingSquares))
	}

	return moves, captures, triggerPromotion, checkCount > 0
}

func GetValidMovesForPiece(piecePosition int, currentGameState gameState) (moves []int, captures []int, triggerPromotion bool, friendlyKingInCheck bool) {
	moves, captures, triggerPromotion, friendlyKingInCheck = getMovesandCapturesForPiece(piecePosition, currentGameState)
	return
}

func IsMoveValid(fen string, piece int, move int) bool {
	var currentGameState = BoardFromFEN(fen)
	var moves, captures, _, _ = GetValidMovesForPiece(piece, currentGameState)

	for _, possibleMove := range append(moves, captures...) {
		if move == possibleMove {
			return true
		}
	}

	return false
}

func canColourMove(currentGameState gameState, colour pieceColour) bool {
	var piece *pieceType
	for i := range currentGameState.board {
		piece = currentGameState.board[i].piece
		if piece != nil && piece.colour == colour {
			var moves, captures, _, _ = GetValidMovesForPiece(i, currentGameState)
			if len(moves) > 0 || len(captures) > 0 {
				return true
			}
		}
	}
	return false
}
