import { defineConfig } from '@playwright/test';

const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';

export default defineConfig({
  testDir: 'tests',
  use: { baseURL: BASE_URL, browserName: 'chromium' },
  webServer: { command: 'npm run dev', url: BASE_URL, reuseExistingServer: true },
  projects: [
    {
      name: 'mobile',
      use: { viewport: { width: 360, height: 740 }, isMobile: true, hasTouch: true }
    },
    { name: 'desktop', use: { viewport: { width: 1280, height: 800 } } }
  ]
});
