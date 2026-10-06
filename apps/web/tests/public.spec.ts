import { expect, test } from '@playwright/test';

function uniquePhone(): string {
  const stamp = Date.now().toString().slice(-6);
  const random = Math.floor(Math.random() * 100)
    .toString()
    .padStart(2, '0');
  return '0857' + stamp + random;
}

test('pelanggan baru memesan dari halaman depot lalu membatalkan pesanannya', async ({ page }) => {
  await page.goto('/d/depot-tirta-sejuk');
  await expect(page.getByRole('heading', { level: 1, name: 'Depot Tirta Sejuk' })).toBeVisible();
  await expect(page.getByText('Menerima pesanan')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Pesan sekarang' })).toBeEnabled();

  await page.getByLabel('Nama', { exact: true }).fill('Pelanggan Uji');
  await page.getByLabel('Nomor WhatsApp').fill(uniquePhone());
  await page.getByLabel('Alamat', { exact: true }).fill('Jl. Uji Otomatis No. 7');
  await page.getByRole('button', { name: 'Tambah' }).click();
  await expect(page.getByRole('group', { name: 'Jumlah galon' })).toContainText('2');
  await page.getByRole('button', { name: 'Pesan sekarang' }).click();

  await expect(page).toHaveURL(/\/t\/[A-Za-z0-9]+$/);
  await expect(page.getByText('Menunggu konfirmasi')).toBeVisible();
  await expect(page.getByText('2 x Isi ulang galon')).toBeVisible();

  await page.getByRole('button', { name: 'Batalkan pesanan' }).click();
  await page.getByRole('button', { name: 'Ya, batalkan' }).click();
  await expect(page.getByText('Dibatalkan').first()).toBeVisible();
  await expect(page.getByRole('button', { name: 'Batalkan pesanan' })).toHaveCount(0);
});

test('link pelacakan yang tidak dikenal menampilkan halaman 404', async ({ page }) => {
  const response = await page.goto('/t/tokentidakada0000');
  expect(response?.status()).toBe(404);
  await expect(page.getByRole('heading', { level: 1, name: 'Link tidak dikenal.' })).toBeVisible();
});
