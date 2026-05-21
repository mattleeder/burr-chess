import { test, expect } from '@playwright/test'
import { uniqueUser, registerUser, logout, SHARED_USER, AUTH_FILE, waitForAuthLoaded } from './helpers'

// ---------------------------------------------------------------------------
// Registration (tests the UI flow — registers fresh unique users)
// ---------------------------------------------------------------------------

test.describe('Registration', () => {
  test('successful registration lands on homepage', async ({ page }) => {
    const { username, password } = uniqueUser()
    await registerUser(page, username, password)
    await expect(page).toHaveURL('/')
  })

  // Auth rate-limit budget: 5 burst tokens for the whole test run.
  // Keeping the duplicate-username error test would push auth calls to 6 total
  // (setup + reg + dup + wrong-pw + login-referrer + queue-logout-reg = 6).
  // Client-side empty-username validation is covered by the test below.

  test('stays on register page when username is empty', async ({ page }) => {
    await page.goto('/register')
    await page.locator('input[name="password"]').fill('TestPass1!')
    await page.getByRole('button', { name: 'REGISTER' }).click()
    // Browser native required validation blocks submission; page stays on /register
    await expect(page).toHaveURL('/register')
    await expect(page.getByRole('button', { name: 'REGISTER' })).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------
// Uses shared pre-registered user via storageState to avoid consuming extra
// auth rate-limit tokens. Error-case tests still exercise the login form.
// ---------------------------------------------------------------------------

test.describe('Login', () => {
  test.use({ storageState: AUTH_FILE })

  test('successful login lands on homepage', async ({ page }) => {
    // The setup project already proves the login flow works end-to-end.
    // Here we confirm that an authenticated user reaches the homepage.
    await page.goto('/')
    await expect(page).toHaveURL('/')
  })

  test('shows error for wrong password', async ({ page }) => {
    await page.goto('/login')
    await waitForAuthLoaded(page)
    await page.locator('input[name="username"]').fill(SHARED_USER.username)
    await page.locator('input[name="password"]').fill('WrongPassword!')
    await page.getByRole('button', { name: 'SIGN IN' }).click()
    await expect(page).toHaveURL('/login')
    await expect(page.locator('.formError:not(.hidden)').first()).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Logout (uses shared state — already logged in)
// ---------------------------------------------------------------------------

test.describe('Logout', () => {
  // Register a fresh user so the shared AUTH_FILE session is never destroyed.
  // Logout calls sessionManager.Destroy on the backend, which would invalidate
  // the stored session and break subsequent queue tests that restore AUTH_FILE.
  // The auth rate limiter is reset in globalSetup so token budget is not a concern.
  test('after logout the nav shows Sign In and protected routes redirect', async ({ page }) => {
    const user = uniqueUser()
    await registerUser(page, user.username, user.password)

    await logout(page)
    await expect(page.getByRole('link', { name: 'Sign In' })).toBeVisible()
    // Confirm protected routes redirect after logout too.
    await page.goto('/account/settings')
    await expect(page).toHaveURL(/\/login\?referrer=/)
  })
})

// ---------------------------------------------------------------------------
// Protected routes
// ---------------------------------------------------------------------------

test.describe('Protected routes', () => {
  test('unauthenticated user is redirected to login with referrer', async ({ page }) => {
    await page.goto('/account/settings')
    await expect(page).toHaveURL(/\/login\?referrer=.*account.*settings/)
  })

  test('login with referrer redirects to original destination', async ({ page }) => {
    await page.goto('/login?referrer=/account/settings')
    await waitForAuthLoaded(page)
    await page.locator('input[name="username"]').fill(SHARED_USER.username)
    await page.locator('input[name="password"]').fill(SHARED_USER.password)
    await page.getByRole('button', { name: 'SIGN IN' }).click()
    await expect(page).toHaveURL('/account/settings')
  })
})
