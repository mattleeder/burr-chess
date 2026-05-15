import { describe, it, expect, vi } from 'vitest'
import {
  GameState,
  GameAction,
  gameReducer,
  dispatchWebSocketMessage,
  createInitialState,
  OpponentEventType,
  boardHistory,
} from './GameContext'
import { PieceColour } from './ChessLogic'

// Real FENs used to drive board-advance tests
const STARTING_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'
const AFTER_E4_FEN = 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
const AFTER_E4_E5_FEN = 'rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2'
const AFTER_NF3_FEN = 'rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2'

function makeEntry(fen: string, lastMove: [number, number] = [0, 0]): boardHistory {
  return {
    FEN: fen,
    lastMove,
    algebraicNotation: '',
    whitePlayerTimeRemainingMilliseconds: 300000,
    blackPlayerTimeRemainingMilliseconds: 300000,
  }
}

// Convenience: dispatch a single ON_MOVE action
function dispatchMove(
  state: GameState,
  fens: string[],
  overrides: Partial<{ gameOverStatus: number; threefoldRepetition: boolean }> = {}
): GameState {
  const action: GameAction = {
    type: 'ON_MOVE',
    body: {
      matchStateHistory: fens.map(makeEntry),
      gameOverStatus: overrides.gameOverStatus ?? 0,
      threefoldRepetition: overrides.threefoldRepetition ?? false,
    },
  }
  return gameReducer(state, action)
}

// ── createInitialState ──────────────────────────────────────────────────────

describe('createInitialState', () => {
  it('creates state with the starting FEN', () => {
    const state = createInitialState(300000)
    expect(state.matchData.activeState.FEN).toBe(STARTING_FEN)
  })

  it('seeds both clocks from the time format argument', () => {
    const state = createInitialState(180000)
    expect(state.matchData.activeState.whitePlayerTimeRemainingMilliseconds).toBe(180000)
    expect(state.matchData.activeState.blackPlayerTimeRemainingMilliseconds).toBe(180000)
  })

  it('defaults playerColour to Spectator', () => {
    expect(createInitialState(300000).playerColour).toBe(PieceColour.Spectator)
  })

  it('sets activeMove to 0', () => {
    expect(createInitialState(300000).matchData.activeMove).toBe(0)
  })

  it('starts with gameOverStatus 0', () => {
    expect(createInitialState(300000).matchData.gameOverStatus).toBe(0)
  })
})

// ── PLAYER_CODE ─────────────────────────────────────────────────────────────

describe('gameReducer — PLAYER_CODE', () => {
  it('assigns White for code 0', () => {
    const next = gameReducer(createInitialState(300000), {
      type: 'PLAYER_CODE',
      body: { playerCode: 0 },
    })
    expect(next.playerColour).toBe(PieceColour.White)
  })

  it('assigns Black for code 1', () => {
    const next = gameReducer(createInitialState(300000), {
      type: 'PLAYER_CODE',
      body: { playerCode: 1 },
    })
    expect(next.playerColour).toBe(PieceColour.Black)
  })

  it('leaves playerColour unchanged for an unknown code', () => {
    const state = createInitialState(300000)
    const next = gameReducer(state, { type: 'PLAYER_CODE', body: { playerCode: 99 } })
    expect(next.playerColour).toBe(state.playerColour)
  })
})

// ── ON_MOVE ──────────────────────────────────────────────────────────────────

describe('gameReducer — ON_MOVE', () => {
  it('advances the board when the user is viewing the latest move', () => {
    const next = dispatchMove(createInitialState(300000), [STARTING_FEN, AFTER_E4_FEN])
    expect(next.matchData.activeState.FEN).toBe(AFTER_E4_FEN)
    expect(next.matchData.activeMove).toBe(1)
  })

  it('keeps the view when the user is browsing history', () => {
    // Build a 3-move game
    const state2 = dispatchMove(
      dispatchMove(createInitialState(300000), [STARTING_FEN, AFTER_E4_FEN]),
      [STARTING_FEN, AFTER_E4_FEN, AFTER_E4_E5_FEN]
    )
    // Simulate the user going back to view move 1
    const browsing: GameState = {
      ...state2,
      matchData: {
        ...state2.matchData,
        activeMove: 1,
        activeState: { ...state2.matchData.activeState, FEN: AFTER_E4_FEN },
      },
    }
    // A new move arrives
    const next = dispatchMove(browsing, [
      STARTING_FEN, AFTER_E4_FEN, AFTER_E4_E5_FEN, AFTER_NF3_FEN,
    ])
    // View must not jump forward
    expect(next.matchData.activeState.FEN).toBe(AFTER_E4_FEN)
    expect(next.matchData.activeMove).toBe(1)
    // But history must be updated
    expect(next.matchData.stateHistory).toHaveLength(4)
  })

  it('returns the same state reference on empty history', () => {
    const state = createInitialState(300000)
    const next = dispatchMove(state, [])
    expect(next).toBe(state)
  })

  it('updates gameOverStatus', () => {
    const next = dispatchMove(createInitialState(300000), [STARTING_FEN, AFTER_E4_FEN], {
      gameOverStatus: 2,
    })
    expect(next.matchData.gameOverStatus).toBe(2)
  })

  it('updates the threefoldRepetition flag', () => {
    const next = dispatchMove(createInitialState(300000), [STARTING_FEN, AFTER_E4_FEN], {
      threefoldRepetition: true,
    })
    expect(next.threefoldRepetition).toBe(true)
  })

  it('clears a pending opponentEventType on each move', () => {
    const withDraw = gameReducer(createInitialState(300000), {
      type: 'OPPONENT_EVENT',
      body: { sender: 'white', eventType: 'draw' },
    })
    expect(withDraw.opponentEventType).toBe(OpponentEventType.Draw)

    const next = dispatchMove(withDraw, [STARTING_FEN, AFTER_E4_FEN])
    expect(next.opponentEventType).toBe(OpponentEventType.None)
  })

  it('updates both clocks from the latest history entry', () => {
    const entry: boardHistory = {
      ...makeEntry(AFTER_E4_FEN),
      whitePlayerTimeRemainingMilliseconds: 295000,
      blackPlayerTimeRemainingMilliseconds: 298000,
    }
    const action: GameAction = {
      type: 'ON_MOVE',
      body: { matchStateHistory: [makeEntry(STARTING_FEN), entry], gameOverStatus: 0, threefoldRepetition: false },
    }
    const next = gameReducer(createInitialState(300000), action)
    expect(next.matchData.activeState.whitePlayerTimeRemainingMilliseconds).toBe(295000)
    expect(next.matchData.activeState.blackPlayerTimeRemainingMilliseconds).toBe(298000)
  })

  it('derives activeColour from the latest FEN', () => {
    // After 1.e4 it is Black's turn
    const next = dispatchMove(createInitialState(300000), [STARTING_FEN, AFTER_E4_FEN])
    expect(next.matchData.activeColour).toBe(PieceColour.Black)
  })
})

// ── ON_CONNECT ───────────────────────────────────────────────────────────────

describe('gameReducer — ON_CONNECT', () => {
  function connectAction(overrides: Partial<{
    whiteConnected: boolean
    blackConnected: boolean
    timeoutMs: number
    whiteUsername: string | null
    blackUsername: string | null
  }> = {}): GameAction {
    return {
      type: 'ON_CONNECT',
      body: {
        matchStateHistory: [makeEntry(STARTING_FEN)],
        gameOverStatus: 0,
        threefoldRepetition: false,
        whitePlayerConnected: overrides.whiteConnected ?? true,
        blackPlayerConnected: overrides.blackConnected ?? true,
        millisecondsUntilTimeout: overrides.timeoutMs ?? 0,
        whitePlayerUsername: overrides.whiteUsername ?? null,
        blackPlayerUsername: overrides.blackUsername ?? null,
      },
    }
  }

  it('sets connection status for both players', () => {
    const next = gameReducer(
      createInitialState(300000),
      connectAction({ whiteConnected: true, blackConnected: false })
    )
    expect(next.isWhiteConnected).toBe(true)
    expect(next.isBlackConnected).toBe(false)
  })

  it('sets player usernames', () => {
    const next = gameReducer(
      createInitialState(300000),
      connectAction({ whiteUsername: 'alice', blackUsername: 'bob' })
    )
    expect(next.whitePlayerUsername).toBe('alice')
    expect(next.blackPlayerUsername).toBe('bob')
  })

  it('sets millisecondsUntilOpponentTimeout when > 0', () => {
    const next = gameReducer(
      createInitialState(300000),
      connectAction({ timeoutMs: 25000 })
    )
    expect(next.millisecondsUntilOpponentTimeout).toBe(25000)
  })

  it('sets millisecondsUntilOpponentTimeout to null when 0', () => {
    const next = gameReducer(
      createInitialState(300000),
      connectAction({ timeoutMs: 0 })
    )
    expect(next.millisecondsUntilOpponentTimeout).toBeNull()
  })
})

// ── CONNECTION_STATUS ────────────────────────────────────────────────────────

describe('gameReducer — CONNECTION_STATUS', () => {
  it('marks white as disconnected', () => {
    const state = { ...createInitialState(300000), isWhiteConnected: true }
    const next = gameReducer(state, {
      type: 'CONNECTION_STATUS',
      body: { playerColour: 'white', isConnected: false, millisecondsUntilTimeout: 30000 },
    })
    expect(next.isWhiteConnected).toBe(false)
  })

  it('marks black as reconnected', () => {
    const state = { ...createInitialState(300000), isBlackConnected: false }
    const next = gameReducer(state, {
      type: 'CONNECTION_STATUS',
      body: { playerColour: 'black', isConnected: true, millisecondsUntilTimeout: 0 },
    })
    expect(next.isBlackConnected).toBe(true)
  })

  it('sets the disconnect timeout', () => {
    const next = gameReducer(createInitialState(300000), {
      type: 'CONNECTION_STATUS',
      body: { playerColour: 'white', isConnected: false, millisecondsUntilTimeout: 30000 },
    })
    expect(next.millisecondsUntilOpponentTimeout).toBe(30000)
  })

  it('clears the timeout on reconnect', () => {
    const state = { ...createInitialState(300000), millisecondsUntilOpponentTimeout: 25000 }
    const next = gameReducer(state, {
      type: 'CONNECTION_STATUS',
      body: { playerColour: 'white', isConnected: true, millisecondsUntilTimeout: 0 },
    })
    expect(next.millisecondsUntilOpponentTimeout).toBeNull()
  })

  it('does not affect the other player when one reconnects', () => {
    const state = { ...createInitialState(300000), isBlackConnected: false }
    const next = gameReducer(state, {
      type: 'CONNECTION_STATUS',
      body: { playerColour: 'white', isConnected: false, millisecondsUntilTimeout: 0 },
    })
    expect(next.isBlackConnected).toBe(false)
  })
})

// ── OPPONENT_EVENT ───────────────────────────────────────────────────────────

describe('gameReducer — OPPONENT_EVENT', () => {
  it('sets Takeback', () => {
    const next = gameReducer(createInitialState(300000), {
      type: 'OPPONENT_EVENT',
      body: { sender: 'white', eventType: 'takeback' },
    })
    expect(next.opponentEventType).toBe(OpponentEventType.Takeback)
  })

  it('sets Draw', () => {
    const next = gameReducer(createInitialState(300000), {
      type: 'OPPONENT_EVENT',
      body: { sender: 'white', eventType: 'draw' },
    })
    expect(next.opponentEventType).toBe(OpponentEventType.Draw)
  })

  it('sets Rematch', () => {
    const next = gameReducer(createInitialState(300000), {
      type: 'OPPONENT_EVENT',
      body: { sender: 'white', eventType: 'rematch' },
    })
    expect(next.opponentEventType).toBe(OpponentEventType.Rematch)
  })

  it('ignores unknown event types', () => {
    const next = gameReducer(createInitialState(300000), {
      type: 'OPPONENT_EVENT',
      body: { sender: 'white', eventType: 'unknownEvent' },
    })
    expect(next.opponentEventType).toBe(OpponentEventType.None)
  })
})

// ── dispatchWebSocketMessage ─────────────────────────────────────────────────

describe('dispatchWebSocketMessage', () => {
  it('dispatches PLAYER_CODE for sendPlayerCode', () => {
    const dispatch = vi.fn()
    dispatchWebSocketMessage(
      JSON.stringify({ messageType: 'sendPlayerCode', body: { playerCode: 0 } }),
      dispatch
    )
    expect(dispatch).toHaveBeenCalledWith({
      type: 'PLAYER_CODE',
      body: { playerCode: 0 },
    })
  })

  it('dispatches ON_CONNECT for onConnect', () => {
    const dispatch = vi.fn()
    const body = {
      matchStateHistory: [],
      gameOverStatus: 0,
      threefoldRepetition: false,
      whitePlayerConnected: true,
      blackPlayerConnected: true,
      millisecondsUntilTimeout: 0,
      whitePlayerUsername: 'alice',
      blackPlayerUsername: 'bob',
    }
    dispatchWebSocketMessage(JSON.stringify({ messageType: 'onConnect', body }), dispatch)
    expect(dispatch).toHaveBeenCalledWith({ type: 'ON_CONNECT', body })
  })

  it('dispatches ON_MOVE for onMove', () => {
    const dispatch = vi.fn()
    const body = { matchStateHistory: [], gameOverStatus: 0, threefoldRepetition: false }
    dispatchWebSocketMessage(JSON.stringify({ messageType: 'onMove', body }), dispatch)
    expect(dispatch).toHaveBeenCalledWith({ type: 'ON_MOVE', body })
  })

  it('dispatches CONNECTION_STATUS for connectionStatus', () => {
    const dispatch = vi.fn()
    const body = { playerColour: 'white', isConnected: false, millisecondsUntilTimeout: 30000 }
    dispatchWebSocketMessage(
      JSON.stringify({ messageType: 'connectionStatus', body }),
      dispatch
    )
    expect(dispatch).toHaveBeenCalledWith({ type: 'CONNECTION_STATUS', body })
  })

  it('dispatches OPPONENT_EVENT for opponentEvent', () => {
    const dispatch = vi.fn()
    const body = { sender: 'white', eventType: 'draw' }
    dispatchWebSocketMessage(
      JSON.stringify({ messageType: 'opponentEvent', body }),
      dispatch
    )
    expect(dispatch).toHaveBeenCalledWith({ type: 'OPPONENT_EVENT', body })
  })

  it('handles multiple newline-separated messages in one call', () => {
    const dispatch = vi.fn()
    const msg =
      JSON.stringify({ messageType: 'sendPlayerCode', body: { playerCode: 0 } }) +
      '\n' +
      JSON.stringify({ messageType: 'sendPlayerCode', body: { playerCode: 1 } })
    dispatchWebSocketMessage(msg, dispatch)
    expect(dispatch).toHaveBeenCalledTimes(2)
  })

  it('does nothing for non-string data', () => {
    const dispatch = vi.fn()
    dispatchWebSocketMessage(42, dispatch)
    expect(dispatch).not.toHaveBeenCalled()
  })

  it('skips malformed JSON without throwing', () => {
    const dispatch = vi.fn()
    expect(() => dispatchWebSocketMessage('not json', dispatch)).not.toThrow()
    expect(dispatch).not.toHaveBeenCalled()
  })

  it('does not dispatch for unknown message types', () => {
    const dispatch = vi.fn()
    dispatchWebSocketMessage(
      JSON.stringify({ messageType: 'unknownType', body: {} }),
      dispatch
    )
    expect(dispatch).not.toHaveBeenCalled()
  })
})
