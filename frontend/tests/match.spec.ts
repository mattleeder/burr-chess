import { test, expect, BrowserContext, Page } from '@playwright/test'
import { waitForAuthLoaded, AUTH_FILE } from './helpers'

// ── Helpers ──

// Click a square by board index (0=a8, 63=h1), accounting for the board flip.
// When flipped, visual square for boardIdx is at 63−boardIdx.
// Uses element-relative click so Playwright handles scroll/viewport offsets.
async function clickSquare(page: Page, boardIdx: number, flipped: boolean) {
  const board = page.locator('.chessboard')
  const box = await board.boundingBox()
  if (!box) throw new Error('Chessboard not found')
  const squareWidth = box.width / 8
  const visualIdx = flipped ? 63 - boardIdx : boardIdx
  const row = Math.floor(visualIdx / 8)
  const col = visualIdx % 8
  await board.click({
    position: {
      x: col * squareWidth + squareWidth / 2,
      y: row * squareWidth + squareWidth / 2,
    },
  })
}

// The white king starts at e1 (board index 60).
// Not flipped (white's view): king is at visual row 7 — Y ≈ 700px on an 800px board.
// Flipped (black's view):     king is at visual row 0 — Y ≈ 0px.
// Threshold of <100px reliably identifies the flipped state.
async function isBoardFlipped(page: Page): Promise<boolean> {
  await page.locator('.white-king').waitFor({ timeout: 5000 })
  return page.evaluate(() => {
    const king = document.querySelector('.white-king') as HTMLElement | null
    if (!king) return false
    const match = king.style.transform.match(/translate\([^,]+,\s*([\d.]+)px\)/)
    return match ? parseFloat(match[1]) < 100 : false
  })
}

// Wait for a page to land in a matchroom, board rendered with pieces, and
// the WebSocket onConnect message received (player names + white king visible).
async function waitForMatchroom(page: Page) {
  await page.waitForURL(/\/matchroom\//, { timeout: 15000 })
  await expect(page.locator('.chessboard')).toBeVisible()
  // .white-king visible means pieces are rendered and onConnect was received.
  await page.locator('.white-king').waitFor({ timeout: 8000 })
}

// Navigate both pages to the homepage, pair them in the 3+0 queue, and wait
// for both to reach the matchroom.
async function setupMatch(page1: Page, page2: Page) {
  await Promise.all([
    page1.goto('/').then(() => waitForAuthLoaded(page1)),
    page2.goto('/').then(() => waitForAuthLoaded(page2)),
  ])
  // Wait for the SSE 200 response — this guarantees the backend has registered
  // the client channel before page2 joins, so matchmaking can't fire and send a
  // match-found notification before page1's SSE handler is ready.
  await Promise.all([
    page1.waitForResponse((r) => r.url().includes('/listenformatch') && r.status() === 200),
    page1.getByRole('button', { name: /3 \+ 0/ }).click(),
  ])
  await page2.getByRole('button', { name: /3 \+ 0/ }).click()
  await Promise.all([waitForMatchroom(page1), waitForMatchroom(page2)])
}

// ── Tests ──

test.describe('Match flow', () => {
  // page1 = shared auth user (storageState), page2 = anonymous.
  // Neither context registers or logs in, so auth rate-limit budget is unchanged.
  let ctx1: BrowserContext
  let ctx2: BrowserContext
  let page1: Page
  let page2: Page

  test.beforeEach(async ({ browser }) => {
    // Clear any live matches left over from prior tests in this run, so the
    // shared auth user is not blocked from queuing again.
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

    ctx1 = await browser.newContext({ storageState: AUTH_FILE })
    page1 = await ctx1.newPage()
    ctx2 = await browser.newContext()
    page2 = await ctx2.newPage()
  })

  test.afterEach(async () => {
    await ctx1.close()
    await ctx2.close()
  })

  test('two players are matched and both reach the same matchroom', async () => {
    await setupMatch(page1, page2)

    const url1 = page1.url()
    const url2 = page2.url()
    expect(url1).toMatch(/\/matchroom\/\d+/)
    expect(url2).toMatch(/\/matchroom\/\d+/)
    expect(url1).toBe(url2)

    // Board, clocks, and player names are all visible.
    await expect(page1.locator('.chessboard')).toBeVisible()
    await expect(page1.locator('.playerTimeTop')).toBeVisible()
    await expect(page1.locator('.playerTimeBottom')).toBeVisible()
    await expect(page1.locator('.playerName').first()).toBeVisible()
  })

  test('white can make the first move', async () => {
    await setupMatch(page1, page2)

    // Determine who is white by where the white king renders.
    const p1Flipped = await isBoardFlipped(page1)
    const whitePage = p1Flipped ? page2 : page1
    const whiteFlipped = false // white's board is never flipped

    // Click e2 (board index 52) — potential-move dots should appear.
    await clickSquare(whitePage, 52, whiteFlipped)
    await expect(whitePage.locator('.potential-move').first()).toBeVisible({ timeout: 8000 })

    // Click e4 (board index 36) — move is sent via WebSocket.
    await clickSquare(whitePage, 36, whiteFlipped)

    // "e4" appears in the move history on both sides.
    await expect(page1.locator('.movesContainer').getByText('e4')).toBeVisible({ timeout: 5000 })
    await expect(page2.locator('.movesContainer').getByText('e4')).toBeVisible({ timeout: 5000 })
  })

  test('a player can resign and both sides see the result', async () => {
    await setupMatch(page1, page2)

    // Flag (resign) is the Lucide svg inside the third .gameControlsButton.
    // The onClick handler is on the svg, not the wrapper div.
    await page1.locator('svg.lucide-flag').click()

    // Both boards show the game-over overlay ("White Resigned" or "Black Resigned").
    await expect(page1.locator('.chessboard').getByText(/Resigned/)).toBeVisible({ timeout: 5000 })
    await expect(page2.locator('.chessboard').getByText(/Resigned/)).toBeVisible({ timeout: 5000 })
  })

  test('draw offer can be declined', async () => {
    await setupMatch(page1, page2)

    // Handshake (draw offer) onClick is on the svg element.
    await page1.locator('svg.lucide-handshake').click()

    // page2 sees the event dialog and clicks Decline.
    await expect(page2.locator('.eventTypeDialog')).toBeVisible({ timeout: 8000 })
    await page2.locator('.eventTypeDialog').getByRole('button', { name: 'Decline' }).click()

    // Dialog disappears — game continues.
    await expect(page2.locator('.eventTypeDialog')).not.toBeVisible({ timeout: 3000 })
    // Game is still ongoing (no game-over text on the board).
    await expect(page1.locator('.chessboard').getByText(/Resigned|Draw|Checkmate/)).not.toBeVisible()
  })
})
