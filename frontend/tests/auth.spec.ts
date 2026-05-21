import { test, expect } from '@playwright/test'
import { uniqueUser, registerUser, loginUser, logout, waitForAuthLoaded } from './helpers'

// ---------------------------------------------------------------------------
// Registration (tests the UI flow — registers fresh unique users)
// ---------------------------------------------------------------------------

test.describe('Registration', () => {
  test('successful registration lands on homepage', async ({ page }) => {
    const { username, password } = uniqueUser()
    await registerUser(page, username, password)
    await expect(page).toHaveURL('/')
  })

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

test.describe('Login', () => {
  test('successful login lands on homepage', async ({ page }) => {
    const user = uniqueUser()
    await registerUser(page, user.username, user.password)
    await page.context().clearCookies()
    await loginUser(page, user.username, user.password)
    await expect(page).toHaveURL('/')
  })

  test('shows error for wrong password', async ({ page }) => {
    const user = uniqueUser()
    await registerUser(page, user.username, user.password)
    await page.context().clearCookies()
    await page.goto('/login')
    await waitForAuthLoaded(page)
    await page.locator('input[name="username"]').fill(user.username)
    await page.locator('input[name="password"]').fill('WrongPassword!')
    await page.getByRole('button', { name: 'SIGN IN' }).click()
    await expect(page).toHaveURL('/login')
    await expect(page.locator('.formError:not(.hidden)').first()).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

test.describe('Logout', () => {
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
    const user = uniqueUser()
    await registerUser(page, user.username, user.password)
    await page.context().clearCookies()
    await page.goto('/login?referrer=/account/settings')
    await waitForAuthLoaded(page)
    await page.locator('input[name="username"]').fill(user.username)
    await page.locator('input[name="password"]').fill(user.password)
    await page.getByRole('button', { name: 'SIGN IN' }).click()
    await expect(page).toHaveURL('/account/settings')
  })
})
