import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { AuthContext, AuthContextType } from './AuthContext'
import { ProtectedRoute } from './ProtectedRoute'

// Rendered inside the router so it always reflects the current location
function LocationDisplay() {
  const { pathname, search } = useLocation()
  return <div data-testid="location">{pathname + search}</div>
}

function renderRoute(
  authOverrides: Partial<AuthContextType>,
  initialPath = '/protected'
) {
  const authValue: AuthContextType = {
    isLoading: false,
    isLoggedIn: false,
    authData: { username: '', csrfToken: '' },
    csrfToken: '',
    register: vi.fn() as AuthContextType['register'],
    login: vi.fn() as AuthContextType['login'],
    logout: vi.fn() as AuthContextType['logout'],
    ...authOverrides,
  }

  render(
    <AuthContext.Provider value={authValue}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route
            path="/protected"
            element={<ProtectedRoute element={<div>Protected Content</div>} />}
          />
          <Route
            path="/account"
            element={<ProtectedRoute element={<div>Account Page</div>} />}
          />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
        <LocationDisplay />
      </MemoryRouter>
    </AuthContext.Provider>
  )
}

describe('ProtectedRoute — loading', () => {
  it('renders nothing while auth is loading', () => {
    renderRoute({ isLoading: true })
    expect(screen.queryByText('Protected Content')).toBeNull()
  })

  it('does not navigate away while loading', () => {
    renderRoute({ isLoading: true })
    expect(screen.getByTestId('location').textContent).toBe('/protected')
  })
})

describe('ProtectedRoute — unauthenticated', () => {
  it('redirects to /login', async () => {
    renderRoute({ isLoading: false, isLoggedIn: false })
    await waitFor(() => expect(screen.getByText('Login Page')).toBeInTheDocument())
  })

  it('includes the current path as the referrer param', async () => {
    renderRoute({ isLoading: false, isLoggedIn: false }, '/protected')
    await waitFor(() =>
      expect(screen.getByTestId('location').textContent).toBe(
        '/login?referrer=/protected'
      )
    )
  })

  it('uses the correct referrer for a different protected path', async () => {
    renderRoute({ isLoading: false, isLoggedIn: false }, '/account')
    await waitFor(() =>
      expect(screen.getByTestId('location').textContent).toBe(
        '/login?referrer=/account'
      )
    )
  })
})

describe('ProtectedRoute — authenticated', () => {
  it('renders the element prop', () => {
    renderRoute({ isLoading: false, isLoggedIn: true })
    expect(screen.getByText('Protected Content')).toBeInTheDocument()
  })

  it('does not navigate away', async () => {
    renderRoute({ isLoading: false, isLoggedIn: true })
    // Give any effect a chance to fire, then confirm location is unchanged
    await waitFor(() =>
      expect(screen.getByTestId('location').textContent).toBe('/protected')
    )
  })
})
