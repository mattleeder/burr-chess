import { Page } from '@playwright/test'

// Shared test user created once in auth.setup.ts and reused via storageState.
export const SHARED_USER = {
  username: 'e2e_shared',
  password: 'SharedPass1!',
}
export const AUTH_FILE = 'playwright/.auth/user.json'

// Each test run gets a unique username suffix so tests don't collide in SQLite.
let counter = 0
export function uniqueUser(): { username: string; password: string } {
  counter++
  const username = `e2e_${Date.now()}_${counter}`
  return { username, password: 'TestPass1!' }
}

// Wait for the AuthContext to finish initialising (validateSession complete, csrfToken set).
// AuthContext sets data-auth-loaded="true" on <body> in its finally block.
export async function waitForAuthLoaded(page: Page): Promise<void> {
  await page.waitForSelector('body[data-auth-loaded="true"]')
}

// Register a new user via the UI and land on the homepage.
export async function registerUser(
  page: Page,
  username: string,
  password: string
): Promise<void> {
  await page.goto('/register')
  await waitForAuthLoaded(page)
  await page.locator('input[name="username"]').fill(username)
  await page.locator('input[name="password"]').fill(password)
  await page.getByRole('button', { name: 'REGISTER' }).click()
  await page.waitForURL('/')
}

// Log in an existing user via the UI and land on the homepage.
export async function loginUser(
  page: Page,
  username: string,
  password: string
): Promise<void> {
  await page.goto('/login')
  await waitForAuthLoaded(page)
  await page.locator('input[name="username"]').fill(username)
  await page.locator('input[name="password"]').fill(password)
  await page.getByRole('button', { name: 'SIGN IN' }).click()
  await page.waitForURL('/')
}

// Open the account dropdown and click "Log Out".
export async function logout(page: Page): Promise<void> {
  // Wait for auth to be initialised so AccountDropdown is rendered before we try to click it.
  await waitForAuthLoaded(page)
  // AccountDropdown is the first .dropdownContainer inside the right nav;
  // the Settings ToggleDropdown is the second one.
  await page.locator('.navBarContainer.right .dropdownContainer').first().locator('button.dropdownTitle').click()
  await page.getByText('Log Out').click()
}
