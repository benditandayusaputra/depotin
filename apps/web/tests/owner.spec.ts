import { expect, test as base, type Browser } from '@playwright/test';

const OWNER_PHONE = '081200000001';
const OWNER_PASSWORD = 'demo-depotin-2026';
const BASE_URL = 'http://localhost:5173';
const LOGIN_ATTEMPTS = 4;

async function loginOnce(browser: Browser): Promise<string> {
  const context = await browser.newContext({ baseURL: BASE_URL });
  try {
    for (let attempt = 1; attempt <= LOGIN_ATTEMPTS; attempt += 1) {
      const response = await context.request.post('/api/v1/auth/login', {
        data: { phone: OWNER_PHONE, password: OWNER_PASSWORD },
        headers: { origin: BASE_URL }
      });
      if (response.ok()) {
        const file = test.info().outputPath('owner-state.json');
        await context.storageState({ path: file });
        return file;
      }
      if (response.status() !== 429 || attempt === LOGIN_ATTEMPTS) {
        throw new Error(`Masuk gagal: ${response.status()} ${await response.text()}`);
      }
      const retryAfter = Number(response.headers()['retry-after'] ?? '15');
      await new Promise((resolve) => setTimeout(resolve, (retryAfter + 1) * 1000));
    }
  } finally {
    await context.close();
  }
  throw new Error('Masuk gagal.');
}

const test = base.extend<object, { ownerState: string }>({
  ownerState: [
    async ({ browser }, use) => {
      await use(await loginOnce(browser));
    },
    { scope: 'worker' }
  ],
  storageState: ({ ownerState }, use) => use(ownerState)
});

test.describe.configure({ mode: 'serial' });

test('alur C: pesanan baru untuk Bu Rina, tugaskan Andi, lalu batalkan', async ({ page }) => {
  await page.goto('/app/pesanan');
  await expect(page.getByRole('heading', { level: 1, name: 'Pesanan' })).toBeVisible();
  await page.getByRole('link', { name: 'Pesanan baru' }).first().click();
  await expect(page).toHaveURL(/\/app\/pesanan\/baru$/);

  await page.getByLabel('Cari pelanggan').fill('Rina');
  await page.getByRole('button', { name: /Bu Rina/ }).click();
  await expect(page.getByText('Bu Rina')).toBeVisible();
  await page.getByRole('spinbutton', { name: 'Jumlah galon' }).fill('2');
  await page.getByRole('button', { name: 'Simpan pesanan' }).click();

  await expect(page).toHaveURL(/\/app\/pesanan\/[0-9a-f-]{36}$/);
  await expect(page.getByText('Dikonfirmasi', { exact: true }).first()).toBeVisible();
  await expect(page.getByLabel('Total pesanan')).toHaveText(/^Rp\d/);
  await expect(page.getByText('Isi ulang galon × 2')).toBeVisible();

  await page.getByRole('button', { name: 'Tugaskan kurir' }).click();
  await page.getByLabel('Kurir').selectOption({ label: 'Andi' });
  await page.getByRole('button', { name: 'Simpan kurir' }).click();
  await expect(page.getByRole('definition').filter({ hasText: 'Andi' })).toBeVisible();

  await page.getByRole('button', { name: 'Batalkan' }).click();
  await page.getByLabel('Alasan pembatalan').fill('uji otomatis');
  await page.getByRole('button', { name: 'Ya, batalkan pesanan' }).click();
  await expect(page.getByText('Dibatalkan', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('uji otomatis')).toBeVisible();
});

test('pengingat menampilkan antrean dengan tombol Kirim WA', async ({ page }) => {
  await page.goto('/app/pengingat');
  await expect(page.getByRole('heading', { level: 1, name: 'Pengingat' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Kirim WA' }).first()).toBeVisible();
});

test('pelanggan: cari Rina dan lihat kartu prediksi', async ({ page }) => {
  await page.goto('/app/pelanggan');
  await page.getByLabel('Cari pelanggan').fill('Rina');
  await page.getByRole('link', { name: /Bu Rina/ }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Bu Rina' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Perkiraan galon habis' })).toBeVisible();
  await expect(page.getByText(/Biasanya 1 galon habis dalam/)).toBeVisible();
  await expect(page.getByText(/^Keyakinan /)).toBeVisible();
});
