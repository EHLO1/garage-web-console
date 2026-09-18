import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/browser',
  fullyParallel: true,
  use: {
    baseURL: 'http://127.0.0.1:4173',
    browserName: 'chromium',
    channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
    headless: true,
    trace: 'retain-on-failure'
  },
  webServer: {
    command: 'node tests/start-ui.mjs',
    url: 'http://127.0.0.1:4173',
    timeout: 120_000
  },
  reporter: 'list'
});
