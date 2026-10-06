import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: 'screenshots.spec.ts',
  workers: 1,
  timeout: 120_000,
  use: {
    baseURL: 'http://localhost:5173',
    browserName: 'chromium',
    deviceScaleFactor: 2,
    colorScheme: 'light',
    locale: 'id-ID'
  },
  projects: [{ name: 'chromium' }]
});
