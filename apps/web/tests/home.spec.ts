import { expect, test } from '@playwright/test';

test('beranda menampilkan judul Depotin', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1, name: 'Depotin' })).toBeVisible();
});
