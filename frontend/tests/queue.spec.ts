import { test, expect, Page } from '@playwright/test'
import { logout, waitForAuthLoaded, registerUser, uniqueUser, AUTH_FILE } from './helpers'

// All queue tests that need an authenticated user load the shared auth state.
// This avoids extra register calls that would hit the auth rate limit.

test.describe('Queue tiles (unauthenticated)', () => {
  test('queue tiles are visible on the homepage', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByText('Select Time Format')).toBeVisible()
    await expect(page.getByRole('button', { name: /3 \+ 0/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /10 \+ 0/ })).toBeVisible()
  })

  test('unauthenticated users can see all queue tiles', async ({ page }) => {
    await page.goto('/')
    const tileNames = ['1 + 0', '3 + 0', '5 + 0', '10 + 0', '15 + 10']
    for (const name of tileNames) {
      await expect(page.getByRole('button', { name: new RegExp(name.replace('+', '\\+')) })).toBeVisible()
    }
  })
})

test.describe('Queue tiles (authenticated)', () => {
  test.use({ storageState: AUTH_FILE })

  test.beforeEach(async ({ page }) => {
    // Reset backend state so stale live matches from prior tests don't block joinQueue.
    const backendURL = process.env.BACKEND_URL ?? 'http://localhost:8080'
    const resets = await Promise.all([
      fetch(`${backendURL}/resetQueues`),
      fetch(`${backendURL}/resetLiveMatches`),
      fetch(`${backendURL}/resetMatchClients`),
      fetch(`${backendURL}/resetRateLimiters`),
    ])
    for (const r of resets) {
      if (!r.ok) throw new Error(`Reset failed: ${r.url} → ${r.status}`)
    }

    await page.goto('/')
    await waitForAuthLoaded(page)
  })

  // Helper: click a queue button and wait for the SSE connection to be established
  // (200 response from /listenformatch) before asserting the spinner. Without this
  // gate, React 18 batching can process setInQueue(true) and the SSE onerror's
  // setInQueue(false) in the same flush if the backend responds near-instantly,
  // causing the spinner to never appear in the DOM.
  async function joinQueueAndWaitForSSE(page: Page, queueName: RegExp) {
    await Promise.all([
      page.waitForResponse((r: { url(): string; status(): number }) => r.url().includes('/listenformatch') && r.status() === 200),
      page.getByRole('button', { name: queueName }).click(),
    ])
  }

  test('clicking a queue button shows the loading spinner', async ({ page }) => {
    await joinQueueAndWaitForSSE(page, /3 \+ 0/)
    await expect(page.locator('.loaderSpin')).toBeVisible()
    // Leave queue to avoid ghost players in subsequent tests.
    await page.locator('.queueButton:has(.loaderSpin)').click()
    await expect(page.locator('.loaderSpin')).not.toBeVisible({ timeout: 5000 })
  })

  test('clicking the active queue button again leaves the queue', async ({ page }) => {
    await joinQueueAndWaitForSSE(page, /3 \+ 0/)
    await expect(page.locator('.loaderSpin')).toBeVisible()

    // After joining, the button shows a spinner instead of its label.
    // Click the spinning button directly to leave the queue.
    await page.locator('.queueButton:has(.loaderSpin)').click()
    await expect(page.locator('.loaderSpin')).not.toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('button', { name: /3 \+ 0/ })).toBeVisible()
  })

  test('switching queues shows spinner on the new queue', async ({ page }) => {
    await joinQueueAndWaitForSSE(page, /3 \+ 0/)
    await expect(page.locator('.loaderSpin')).toBeVisible()

    // Wait for the new SSE to be established — this is the deterministic signal
    // that the full leave→join→setQueueName sequence has completed and React
    // has re-rendered, moving the spinner from 3+0 to 5+0.
    await Promise.all([
      page.waitForResponse((r: { url(): string; status(): number }) => r.url().includes('/listenformatch') && r.status() === 200),
      page.getByRole('button', { name: /5 \+ 0/ }).click(),
    ])
    // 3+0 tile is back to its label; 5+0 tile shows the spinner.
    await expect(page.getByRole('button', { name: /3 \+ 0/ })).toBeVisible()
    // Leave the 5+0 queue to avoid ghost players in subsequent tests.
    await page.locator('.queueButton:has(.loaderSpin)').click()
    await expect(page.locator('.loaderSpin')).not.toBeVisible({ timeout: 5000 })
  })

  test('navigating away and back resets the queue spinner', async ({ page }) => {
    await joinQueueAndWaitForSSE(page, /3 \+ 0/)
    await expect(page.locator('.loaderSpin')).toBeVisible()

    // Navigate away — QueueTiles unmounts and fires the leave-queue POST.
    // Wait for that POST to complete before full-reloading so the server removes
    // the player from the queue before the next test re-joins.
    await Promise.all([
      page.waitForResponse((r) => r.url().includes('/joinQueue') && r.request().method() === 'POST'),
      page.getByRole('link', { name: 'Watch' }).click(),
    ])
    await expect(page).toHaveURL('/watch')

    // Full reload — QueueTiles remounts with fresh state (no spinner).
    await page.goto('/')
    await expect(page.locator('.loaderSpin')).not.toBeVisible({ timeout: 5000 })
  })

  test('logging out while in queue clears the nav', async ({ page }) => {
    // Register a fresh user so this test is independent of any prior logout that
    // may have invalidated the shared storageState session (logout deletes the
    // server-side session, so AccountDropdown wouldn't render for a stale cookie).
    const user = uniqueUser()
    await registerUser(page, user.username, user.password)

    await page.getByRole('button', { name: /3 \+ 0/ }).click()
    await expect(page.locator('.loaderSpin')).toBeVisible()

    await logout(page)
    await expect(page.getByRole('link', { name: 'Sign In' })).toBeVisible()
  })
})
