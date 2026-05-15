import { describe, it, expect } from 'vitest'
import {
  PieceColour,
  PieceVariant,
  GameOverStatus,
  gameOverDisplayNames,
  parseGameStateFromFEN,
  gameStateToFEN,
} from './ChessLogic'

// FEN for the standard starting position
const STARTING_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

describe('gameOverDisplayNames', () => {
  it('has an entry for every GameOverStatus value', () => {
    const statuses = Object.values(GameOverStatus).filter(
      (v) => typeof v === 'number'
    ) as GameOverStatus[]

    for (const status of statuses) {
      expect(gameOverDisplayNames[status]).toBeDefined()
      expect(typeof gameOverDisplayNames[status]).toBe('string')
    }
  })

  it('maps known statuses to their display strings', () => {
    expect(gameOverDisplayNames[GameOverStatus.Ongoing]).toBe('Ongoing')
    expect(gameOverDisplayNames[GameOverStatus.Checkmate]).toBe('Checkmate')
    expect(gameOverDisplayNames[GameOverStatus.Stalemate]).toBe('Stalemate')
    expect(gameOverDisplayNames[GameOverStatus.ThreefoldRepetition]).toBe('Threefold Repetition')
    expect(gameOverDisplayNames[GameOverStatus.InsufficientMaterial]).toBe('Insufficient Material')
    expect(gameOverDisplayNames[GameOverStatus.WhiteFlagged]).toBe('White Flagged')
    expect(gameOverDisplayNames[GameOverStatus.BlackFlagged]).toBe('Black Flagged')
    expect(gameOverDisplayNames[GameOverStatus.Draw]).toBe('Draw')
    expect(gameOverDisplayNames[GameOverStatus.WhiteResigned]).toBe('White Resigned')
    expect(gameOverDisplayNames[GameOverStatus.BlackResigned]).toBe('Black Resigned')
    expect(gameOverDisplayNames[GameOverStatus.WhiteDisconnected]).toBe('White Disconnected')
    expect(gameOverDisplayNames[GameOverStatus.BlackDisconnected]).toBe('Black Disconnected')
    expect(gameOverDisplayNames[GameOverStatus.GameAborted]).toBe('Game Aborted')
  })
})

describe('parseGameStateFromFEN', () => {
  it('parses the starting position board into 64 squares', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    expect(state.board).toHaveLength(64)
  })

  it('stores the original FEN string', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    expect(state.fen).toBe(STARTING_FEN)
  })

  it('parses white as active colour in starting position', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    expect(state.activeColour).toBe(PieceColour.White)
  })

  it('parses black as active colour when FEN says b', () => {
    const fen = 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
    const state = parseGameStateFromFEN(fen)
    expect(state.activeColour).toBe(PieceColour.Black)
  })

  it('parses all four castling rights in starting position', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    expect(state.whiteCanKingSideCastle).toBe(true)
    expect(state.whiteCanQueenSideCastle).toBe(true)
    expect(state.blackCanKingSideCastle).toBe(true)
    expect(state.blackCanQueenSideCastle).toBe(true)
  })

  it('parses partial castling rights', () => {
    const fen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w Kq - 0 1'
    const state = parseGameStateFromFEN(fen)
    expect(state.whiteCanKingSideCastle).toBe(true)
    expect(state.whiteCanQueenSideCastle).toBe(false)
    expect(state.blackCanKingSideCastle).toBe(false)
    expect(state.blackCanQueenSideCastle).toBe(true)
  })

  it('parses no castling rights when field is -', () => {
    const fen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w - - 0 1'
    const state = parseGameStateFromFEN(fen)
    expect(state.whiteCanKingSideCastle).toBe(false)
    expect(state.whiteCanQueenSideCastle).toBe(false)
    expect(state.blackCanKingSideCastle).toBe(false)
    expect(state.blackCanQueenSideCastle).toBe(false)
  })

  it('parses no en passant square when field is -', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    expect(state.enPassantSquare).toBeNull()
  })

  it('parses en passant square e3 correctly', () => {
    // After 1.e4, the en passant square is e3: file e=4, rank 3 → 4 + (8-3)*8 = 44
    const fen = 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
    const state = parseGameStateFromFEN(fen)
    expect(state.enPassantSquare).toBe(44)
  })

  it('parses en passant square a6 correctly', () => {
    // file a=0, rank 6 → 0 + (8-6)*8 = 16
    const fen = 'rnbqkbnr/1ppppppp/8/pP6/8/8/P1PPPPPP/RNBQKBNR w KQkq a6 0 1'
    const state = parseGameStateFromFEN(fen)
    expect(state.enPassantSquare).toBe(16)
  })

  it('parses the starting position pieces correctly', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    // Index 0 = a8 = black rook
    expect(state.board[0]).toEqual([PieceColour.Black, PieceVariant.Rook])
    // Index 4 = e8 = black king
    expect(state.board[4]).toEqual([PieceColour.Black, PieceVariant.King])
    // Index 8 = a7 = black pawn
    expect(state.board[8]).toEqual([PieceColour.Black, PieceVariant.Pawn])
    // Index 32 = a5 = empty
    expect(state.board[32]).toEqual([null, null])
    // Index 48 = a2 = white pawn
    expect(state.board[48]).toEqual([PieceColour.White, PieceVariant.Pawn])
    // Index 60 = e1 = white king
    expect(state.board[60]).toEqual([PieceColour.White, PieceVariant.King])
    // Index 63 = h1 = white rook
    expect(state.board[63]).toEqual([PieceColour.White, PieceVariant.Rook])
  })

  it('parses empty ranks correctly', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    // Ranks 3-6 (indices 16-47) are empty
    for (let i = 16; i < 48; i++) {
      expect(state.board[i]).toEqual([null, null])
    }
  })
})

describe('gameStateToFEN', () => {
  it('serializes the starting position consistently', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    const result = gameStateToFEN(state)
    // The serializer appends "/" after every 8 squares (including the last rank)
    // and always writes "0 1" for half/full move clocks
    expect(result).toBe('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR/ w KQkq - 0 1')
  })

  it('serializes black to move correctly', () => {
    const fen = 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
    const state = parseGameStateFromFEN(fen)
    const result = gameStateToFEN(state)
    expect(result).toContain(' b ')
  })

  it('serializes partial castling rights correctly', () => {
    const fen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w Kq - 0 1'
    const state = parseGameStateFromFEN(fen)
    const result = gameStateToFEN(state)
    expect(result).toContain('Kq')
  })

  it('serializes no castling rights as -', () => {
    const fen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w - - 0 1'
    const state = parseGameStateFromFEN(fen)
    const result = gameStateToFEN(state)
    expect(result).toContain(' - ')
  })

  it('serializes an en passant square correctly', () => {
    const fen = 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
    const state = parseGameStateFromFEN(fen)
    const result = gameStateToFEN(state)
    expect(result).toContain('e3')
  })

  it('serializes no en passant as -', () => {
    const state = parseGameStateFromFEN(STARTING_FEN)
    const result = gameStateToFEN(state)
    // The en passant field at the end should be " -"
    expect(result).toMatch(/ - 0 1$/)
  })
})
