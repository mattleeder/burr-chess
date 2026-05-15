import { describe, it, expect } from 'vitest'
import { PieceColour } from './ChessLogic'
import { formatDuration, isClockPaused } from './GameInfoTile'
import type { gameContext, matchData } from './GameContext'

// ── formatDuration ───────────────────────────────────────────────────────────

describe('formatDuration', () => {
  it('returns the zero string for 0 ms', () => {
    expect(formatDuration(0)).toBe('00:00:00.0')
  })

  it('returns the zero string for negative values', () => {
    expect(formatDuration(-1)).toBe('00:00:00.0')
    expect(formatDuration(-9999)).toBe('00:00:00.0')
  })

  it('shows tenths for durations under 10 seconds', () => {
    expect(formatDuration(100)).toBe('00:00.1')    // 100ms  → tenths = 1
    expect(formatDuration(1500)).toBe('00:01.5')   // 1.5s   → tenths = 5
    expect(formatDuration(9000)).toBe('00:09.0')   // 9.0s   → tenths = 0
    expect(formatDuration(9999)).toBe('00:09.9')   // 9.999s → tenths = 9
  })

  it('omits tenths for durations of exactly 10 seconds', () => {
    expect(formatDuration(10000)).toBe('00:10')
  })

  it('omits tenths for durations over 10 seconds', () => {
    expect(formatDuration(15750)).toBe('00:15')
  })

  it('formats whole minutes correctly', () => {
    expect(formatDuration(60000)).toBe('01:00')
    expect(formatDuration(300000)).toBe('05:00')
  })

  it('formats minutes and seconds correctly', () => {
    expect(formatDuration(75000)).toBe('01:15')
    expect(formatDuration(185000)).toBe('03:05')
  })

  it('handles large values without tenths', () => {
    expect(formatDuration(600000)).toBe('10:00')
    expect(formatDuration(3599000)).toBe('59:59')
  })
})

// ── isClockPaused ────────────────────────────────────────────────────────────

// Build the minimal matchData shape that isClockPaused reads
function makeMatchData(overrides: {
  gameOverStatus?: number
  activeColour?: PieceColour
  historyLength?: number
}): matchData {
  return {
    gameOverStatus: overrides.gameOverStatus ?? 0,
    activeColour: overrides.activeColour ?? PieceColour.White,
    stateHistory: Array(overrides.historyLength ?? 3).fill({}),
    // Unused by isClockPaused but required by the interface
    activeState: {} as matchData['activeState'],
    activeMove: 0,
  }
}

function makeGame(overrides: Parameters<typeof makeMatchData>[0] = {}): gameContext {
  return { matchData: makeMatchData(overrides) } as gameContext
}

describe('isClockPaused', () => {
  it('is paused when the game is over', () => {
    const game = makeGame({ gameOverStatus: 2, activeColour: PieceColour.White })
    expect(isClockPaused(game, PieceColour.White)).toBe(true)
  })

  it('is paused when it is the other player\'s turn', () => {
    // activeColour is White, checking Black's clock → paused
    const game = makeGame({ activeColour: PieceColour.White })
    expect(isClockPaused(game, PieceColour.Black)).toBe(true)
  })

  it('is paused when fewer than 3 history entries exist (opening)', () => {
    // 2 entries means only white has moved — black's clock hasn't started
    const game = makeGame({ activeColour: PieceColour.Black, historyLength: 2 })
    expect(isClockPaused(game, PieceColour.Black)).toBe(true)
  })

  it('is running when it is this player\'s turn, game is ongoing, and both have moved', () => {
    // 3+ entries: initial + white's move + black's move
    const game = makeGame({ activeColour: PieceColour.White, historyLength: 3 })
    expect(isClockPaused(game, PieceColour.White)).toBe(false)
  })

  it('is running for black when it is black\'s turn with sufficient history', () => {
    const game = makeGame({ activeColour: PieceColour.Black, historyLength: 4 })
    expect(isClockPaused(game, PieceColour.Black)).toBe(false)
  })
})
