import { defineConfig, devices } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

// this file lives at <repo>/external/react/playwright.config.ts
const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // WebGL under headless: SwiftShader software rendering
        launchOptions: {
          args: [
            '--enable-unsafe-swiftshader',
            '--use-gl=angle',
            '--use-angle=swiftshader',
            '--ignore-gpu-blocklist',
          ],
        },
      },
    },
    {
      name: 'mobile',
      use: { ...devices['iPhone 13'] },
    },
  ],
  // The app needs both the vite dev server and the Go backend (the dev
  // proxy forwards /v1 to :1133). Let Playwright own both so the suite is
  // self-contained instead of depending on a manually started server.
  webServer: [
    {
      command: 'npm run dev',
      url: 'http://localhost:5173',
      reuseExistingServer: !process.env.CI,
      timeout: 120 * 1000,
    },
    {
      command: 'go run ./cmd/khayal start',
      cwd: repoRoot,
      // served unauthenticated, so it works as a readiness probe
      url: 'http://localhost:1133/',
      reuseExistingServer: !process.env.CI,
      timeout: 120 * 1000,
      env: { KHAYAL_CONFIG: path.join(repoRoot, 'testdata/config.yaml') },
    },
  ],
})
