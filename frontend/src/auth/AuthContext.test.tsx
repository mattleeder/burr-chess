import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useContext } from 'react'
import { AuthContext, AuthProvider } from './AuthContext'

// Wrapper so renderHook uses AuthProvider
const wrapper = ({ children }: { children: React.ReactNode }) => (
  <AuthProvider>{children}</AuthProvider>
)

function makeFetchResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('AuthProvider — session validation on mount', () => {
  it('sets isLoggedIn and authData on a 200 response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      makeFetchResponse(200, { username: 'alice', csrfToken: 'tok123' })
    ))

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.isLoggedIn).toBe(true)
    expect(result.current.authData.username).toBe('alice')
    expect(result.current.csrfToken).toBe('tok123')
  })

  it('stays logged out but captures csrfToken on a 401 response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      makeFetchResponse(401, { csrfToken: 'anon456' })
    ))

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.isLoggedIn).toBe(false)
    expect(result.current.csrfToken).toBe('anon456')
    expect(result.current.authData.username).toBe('')
  })

  it('stays logged out and finishes loading when fetch throws', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Network error')))

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.isLoggedIn).toBe(false)
    expect(result.current.csrfToken).toBe('')
  })

  it('starts with isLoading:true before the request resolves', () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => { /* never resolves */ })))

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })

    expect(result.current.isLoading).toBe(true)
  })
})

describe('AuthProvider — login', () => {
  it('returns {ok:false} immediately when no csrfToken is set', async () => {
    // validateSession returns a 401 with no token, so csrfToken stays ""
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      makeFetchResponse(401, { csrfToken: '' })
    ))

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    let loginResult
    await act(async () => {
      loginResult = await result.current.login({ username: 'alice', password: 'pw', rememberMe: false })
    })

    expect(loginResult).toEqual({ ok: false })
  })

  it('sets isLoggedIn and updates authData on a successful login', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    // First call: validateSession (401 — logged out but gives CSRF token)
    fetchMock.mockResolvedValueOnce(makeFetchResponse(401, { csrfToken: 'csrf-tok' }))
    // Second call: POST /login → success
    fetchMock.mockResolvedValueOnce(
      makeFetchResponse(200, { username: 'alice', csrfToken: 'new-tok' })
    )

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.csrfToken).toBe('csrf-tok')

    await act(async () => {
      await result.current.login({ username: 'alice', password: 'pw', rememberMe: false })
    })

    expect(result.current.isLoggedIn).toBe(true)
    expect(result.current.authData.username).toBe('alice')
    expect(result.current.csrfToken).toBe('new-tok')
  })

  it('leaves isLoggedIn false on a failed login', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    fetchMock.mockResolvedValueOnce(makeFetchResponse(401, { csrfToken: 'csrf-tok' }))
    fetchMock.mockResolvedValueOnce(
      makeFetchResponse(422, { username: '', password: 'invalid credentials' })
    )

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    let loginResult
    await act(async () => {
      loginResult = await result.current.login({ username: 'alice', password: 'wrong', rememberMe: false })
    })

    expect(result.current.isLoggedIn).toBe(false)
    expect(loginResult).toMatchObject({ ok: false })
  })
})

describe('AuthProvider — register', () => {
  it('returns {ok:false} immediately when no csrfToken is set', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      makeFetchResponse(401, { csrfToken: '' })
    ))

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    let registerResult
    await act(async () => {
      registerResult = await result.current.register({
        username: 'alice', password: 'pw', rememberMe: false,
      })
    })

    expect(registerResult).toEqual({ ok: false })
  })

  it('sets isLoggedIn on successful registration', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    fetchMock.mockResolvedValueOnce(makeFetchResponse(401, { csrfToken: 'csrf-tok' }))
    fetchMock.mockResolvedValueOnce(
      makeFetchResponse(200, { username: 'alice', csrfToken: 'new-tok' })
    )

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => {
      await result.current.register({ username: 'alice', password: 'pw', rememberMe: false })
    })

    expect(result.current.isLoggedIn).toBe(true)
    expect(result.current.authData.username).toBe('alice')
  })
})

describe('AuthProvider — logout', () => {
  it('clears auth state on successful logout', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    // validateSession → logged in
    fetchMock.mockResolvedValueOnce(
      makeFetchResponse(200, { username: 'alice', csrfToken: 'tok' })
    )
    // logout → success (empty body)
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.reject(new SyntaxError('no body')),
    })

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })
    await waitFor(() => expect(result.current.isLoggedIn).toBe(true))

    await act(async () => {
      await result.current.logout()
    })

    expect(result.current.isLoggedIn).toBe(false)
    expect(result.current.authData.username).toBe('')
  })

  it('keeps auth state when logout request fails', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    fetchMock.mockResolvedValueOnce(
      makeFetchResponse(200, { username: 'alice', csrfToken: 'tok' })
    )
    fetchMock.mockResolvedValueOnce(makeFetchResponse(500, {}))

    const { result } = renderHook(() => useContext(AuthContext), { wrapper })
    await waitFor(() => expect(result.current.isLoggedIn).toBe(true))

    await act(async () => {
      await result.current.logout()
    })

    expect(result.current.isLoggedIn).toBe(true)
    expect(result.current.authData.username).toBe('alice')
  })
})
