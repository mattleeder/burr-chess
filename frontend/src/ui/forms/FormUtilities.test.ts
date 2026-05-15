import { describe, it, expect, vi, beforeEach } from 'vitest'
import { submitFormData } from './FormUtilities'

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('submitFormData', () => {
  it('returns ok:true with parsed JSON on a 200 response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ csrfToken: 'abc123', username: 'alice' }),
    }))

    const result = await submitFormData('http://localhost/api/login')

    expect(result.ok).toBe(true)
    expect(result.data).toEqual({ csrfToken: 'abc123', username: 'alice' })
    expect(result.networkError).toBeUndefined()
  })

  it('returns ok:false with parsed JSON on a 4xx response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ username: 'username is required' }),
    }))

    const result = await submitFormData('http://localhost/api/register')

    expect(result.ok).toBe(false)
    expect(result.data).toEqual({ username: 'username is required' })
  })

  it('returns ok:true with no data when response has no JSON body', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.reject(new SyntaxError('No JSON')),
    }))

    const result = await submitFormData('http://localhost/api/logout')

    expect(result.ok).toBe(true)
    expect(result.data).toBeUndefined()
  })

  it('returns networkError:true when fetch throws', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    const result = await submitFormData('http://localhost/api/login')

    expect(result.ok).toBe(false)
    expect(result.networkError).toBe(true)
  })

  it('sends a POST request with provided options', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    })
    vi.stubGlobal('fetch', fetchMock)

    await submitFormData('http://localhost/api/login', {
      headers: { 'X-CSRF-Token': 'tok' },
      body: JSON.stringify({ username: 'alice', password: 'pw' }),
    })

    expect(fetchMock).toHaveBeenCalledOnce()
    const [requestArg, optionsArg] = fetchMock.mock.calls[0]
    expect(requestArg).toBeInstanceOf(Request)
    expect((requestArg as Request).method).toBe('POST')
    expect(optionsArg?.headers).toEqual({ 'X-CSRF-Token': 'tok' })
  })
})
