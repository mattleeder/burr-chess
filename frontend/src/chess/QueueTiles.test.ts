import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiFetch, API } from '../api'
import { queueObjectsMap, toggleQueue, tryLeaveQueue, QueueState } from './QueueTiles'

vi.mock('../api', () => ({
  API: {
    joinQueue: 'http://localhost/api/joinQueue',
    listenForMatch: 'http://localhost/api/listenformatch',
  },
  apiFetch: vi.fn(),
}))

const mockApiFetch = vi.mocked(apiFetch)

// Minimal EventSource stub — must be a real constructor so `new EventSource(...)` works.
const mockEsClose = vi.fn()
class FakeEventSource {
  close = mockEsClose
  onmessage = null
  onerror = null
}

beforeEach(() => {
  mockApiFetch.mockResolvedValue({ ok: true } as Response)
  vi.stubGlobal('EventSource', FakeEventSource)
  mockEsClose.mockClear()
  mockApiFetch.mockClear()
})

// Helper: builds a QueueState with vi.fn() setters.
// The object is cast because test stubs are structurally compatible
// with the React dispatch and ref types but don't match exactly.
function makeQueueState(overrides: Partial<{
  waiting: boolean
  inQueue: boolean
  queueName: string
  error: string
  csrfToken: string
}> = {}): QueueState & {
  setWaiting: ReturnType<typeof vi.fn>
  setInQueue: ReturnType<typeof vi.fn>
  setQueueName: ReturnType<typeof vi.fn>
  setError: ReturnType<typeof vi.fn>
  navigate: ReturnType<typeof vi.fn>
} {
  return {
    waiting: false,
    inQueue: false,
    queueName: '',
    error: '',
    csrfToken: 'test-csrf',
    eventSource: { current: null },
    matchFoundState: { current: null },
    navigate: vi.fn(),
    setWaiting: vi.fn(),
    setInQueue: vi.fn(),
    setQueueName: vi.fn(),
    setError: vi.fn(),
    ...overrides,
  } as unknown as ReturnType<typeof makeQueueState>
}

// ── queueObjectsMap ──────────────────────────────────────────────────────────

describe('queueObjectsMap', () => {
  it('contains all 9 queue names', () => {
    const expected = [
      '1 + 0', '2 + 1', '3 + 0',
      '3 + 2', '5 + 0', '5 + 3',
      '10 + 0', '10 + 5', '15 + 10',
    ]
    for (const key of expected) {
      expect(queueObjectsMap.has(key), `missing key "${key}"`).toBe(true)
    }
    expect(queueObjectsMap.size).toBe(9)
  })

  it.each([
    ['1 + 0',    60_000],
    ['3 + 0',   180_000],
    ['5 + 0',   300_000],
    ['10 + 0',  600_000],
    ['15 + 10', 900_000],
  ])('%s → timeFormat %i ms', (key, ms) => {
    expect(queueObjectsMap.get(key)?.timeFormatInMilliseconds).toBe(ms)
  })

  it.each([
    ['1 + 0',        0],
    ['2 + 1',     1000],
    ['3 + 2',     2000],
    ['5 + 3',     3000],
    ['10 + 5',    5000],
    ['15 + 10', 10_000],
  ])('%s → increment %i ms', (key, ms) => {
    expect(queueObjectsMap.get(key)?.incrementInMilliseconds).toBe(ms)
  })
})

// ── toggleQueue — waiting guard ──────────────────────────────────────────────

describe('toggleQueue — waiting guard', () => {
  it('returns immediately with no side effects when already waiting', async () => {
    const state = makeQueueState({ waiting: true })
    await toggleQueue(state, '3 + 0')
    expect(state.setWaiting).not.toHaveBeenCalled()
    expect(state.setInQueue).not.toHaveBeenCalled()
    expect(state.setQueueName).not.toHaveBeenCalled()
    expect(mockApiFetch).not.toHaveBeenCalled()
  })
})

// ── toggleQueue — join ───────────────────────────────────────────────────────

describe('toggleQueue — join queue', () => {
  it('sets inQueue true and records the queue name', async () => {
    const state = makeQueueState({ inQueue: false })
    await toggleQueue(state, '3 + 0')
    expect(state.setInQueue).toHaveBeenCalledWith(true)
    expect(state.setQueueName).toHaveBeenCalledWith('3 + 0')
  })

  it('sets waiting true then false', async () => {
    const state = makeQueueState({ inQueue: false })
    await toggleQueue(state, '5 + 0')
    expect(state.setWaiting).toHaveBeenNthCalledWith(1, true)
    expect(state.setWaiting).toHaveBeenNthCalledWith(2, false)
  })

  it('clears any previous error at the start', async () => {
    const state = makeQueueState({ inQueue: false, error: 'connection lost' })
    await toggleQueue(state, '3 + 0')
    expect(state.setError).toHaveBeenCalledWith('')
  })

  it('makes exactly one API call (join)', async () => {
    const state = makeQueueState({ inQueue: false })
    await toggleQueue(state, '3 + 0')
    expect(mockApiFetch).toHaveBeenCalledTimes(1)
  })
})

// ── toggleQueue — leave ──────────────────────────────────────────────────────

describe('toggleQueue — leave queue', () => {
  it('sets inQueue false and clears the queue name when clicking the active queue', async () => {
    const state = makeQueueState({ inQueue: true, queueName: '3 + 0' })
    await toggleQueue(state, '3 + 0')
    expect(state.setInQueue).toHaveBeenCalledWith(false)
    expect(state.setQueueName).toHaveBeenCalledWith('')
  })

  it('makes exactly one API call (leave)', async () => {
    const state = makeQueueState({ inQueue: true, queueName: '3 + 0' })
    await toggleQueue(state, '3 + 0')
    expect(mockApiFetch).toHaveBeenCalledTimes(1)
  })
})

// ── toggleQueue — change queue ───────────────────────────────────────────────

describe('toggleQueue — change queue', () => {
  it('updates the queue name to the new queue', async () => {
    const state = makeQueueState({ inQueue: true, queueName: '3 + 0' })
    await toggleQueue(state, '5 + 0')
    expect(state.setQueueName).toHaveBeenCalledWith('5 + 0')
  })

  it('does not change inQueue — user stays in queue', async () => {
    const state = makeQueueState({ inQueue: true, queueName: '3 + 0' })
    await toggleQueue(state, '5 + 0')
    expect(state.setInQueue).not.toHaveBeenCalled()
  })

  it('makes two API calls — leave old queue then join new one', async () => {
    const state = makeQueueState({ inQueue: true, queueName: '3 + 0' })
    await toggleQueue(state, '5 + 0')
    expect(mockApiFetch).toHaveBeenCalledTimes(2)
  })
})

// ── tryLeaveQueue ────────────────────────────────────────────────────────────

describe('tryLeaveQueue', () => {
  it('POSTs the correct timeFormat, increment, and action for the queue', async () => {
    await tryLeaveQueue('3 + 0', { current: null }, 'csrf-tok')
    expect(mockApiFetch).toHaveBeenCalledWith(
      API.joinQueue,
      'csrf-tok',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          timeFormatInMilliseconds: 180_000,
          incrementInMilliseconds: 0,
          action: 'leave',
        }),
      })
    )
  })

  it('sends the right values for a queue with increment', async () => {
    await tryLeaveQueue('5 + 3', { current: null }, 'csrf-tok')
    expect(mockApiFetch).toHaveBeenCalledWith(
      API.joinQueue,
      'csrf-tok',
      expect.objectContaining({
        body: JSON.stringify({
          timeFormatInMilliseconds: 300_000,
          incrementInMilliseconds: 3000,
          action: 'leave',
        }),
      })
    )
  })

  it('closes the EventSource after a successful leave', async () => {
    const mockClose = vi.fn()
    await tryLeaveQueue('3 + 0', { current: { close: mockClose } as unknown as EventSource }, 'csrf-tok')
    expect(mockClose).toHaveBeenCalled()
  })

  it('throws for an unrecognised queue name', async () => {
    await expect(
      tryLeaveQueue('99 + 99', { current: null }, 'csrf')
    ).rejects.toThrow('Queue object not found')
  })

  it('throws when the API returns a non-ok response', async () => {
    mockApiFetch.mockResolvedValueOnce({ ok: false, statusText: 'Forbidden' } as Response)
    await expect(
      tryLeaveQueue('3 + 0', { current: null }, 'csrf')
    ).rejects.toThrow('Forbidden')
  })
})
