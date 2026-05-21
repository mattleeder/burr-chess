import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  fullyParallel: false, // tests share a SQLite DB — run serially
  forbidOnly: !!process.env.CI,
  retries: 1,
  workers: 1,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:5174',
    trace: 'on-first-retry',
    launchOptions: {
      executablePath: process.env.CHROMIUM_PATH,
      args: ['--no-sandbox', '--disable-setuid-sandbox'],
    },
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    // --mode e2e loads .env.e2e which sets VITE_API_URL=/api.
    // The Vite proxy in vite.config.ts then forwards /api/* to http://localhost:8080.
    // The backend must be running at :8080 before starting tests.
    command: 'npm run dev -- --mode e2e --port 5174',
    url: 'http://localhost:5174',
    reuseExistingServer: false,
    timeout: 30_000,
  },
})
