/**
 * Runs once before all E2E tests (via the 'setup' project in playwright.config.ts).
 * Registers a shared test user and saves the authenticated browser state so tests
 * that just need a logged-in user can restore it without going through the UI flow.
 *
 * Strategy: try logging in first (1 auth call). Only register if the user doesn't
 * exist yet (first-ever run). This minimises auth rate-limit consumption across runs.
 */
import { test as setup, expect } from '@playwright/test'
import { SHARED_USER, AUTH_FILE, waitForAuthLoaded } from './helpers'

setup('register and save shared test user', async ({ page }) => {
  // Try logging in first — only 1 auth call if the user already exists.
  await page.goto('/login')
  await waitForAuthLoaded(page)
  await page.locator('input[name="username"]').fill(SHARED_USER.username)
  await page.locator('input[name="password"]').fill(SHARED_USER.password)
  await page.getByRole('button', { name: 'SIGN IN' }).click()

  // If login succeeds the app navigates to '/'; if not (first run, user doesn't exist)
  // we stay on '/login'. Use a short timeout to distinguish the two cases.
  const loginSucceeded = await page.waitForURL('/', { timeout: 5000 }).then(() => true).catch(() => false)

  if (!loginSucceeded) {
    // User does not exist yet (first run) — register instead.
    await page.goto('/register')
    await waitForAuthLoaded(page)
    await page.locator('input[name="username"]').fill(SHARED_USER.username)
    await page.locator('input[name="password"]').fill(SHARED_USER.password)
    await page.getByRole('button', { name: 'REGISTER' }).click()
    await page.waitForURL('/')
  }

  await expect(page).toHaveURL('/')

  // Save authenticated storage state (cookies) for reuse across tests.
  await page.context().storageState({ path: AUTH_FILE })
})
