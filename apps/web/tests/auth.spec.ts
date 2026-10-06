import { expect, test, type Page } from '@playwright/test';

const OWNER_PASSWORD = 'sandi-pemilik-123';
const COURIER_PASSWORD = 'sandi-kurir-12345';

function uniquePhone(prefix: string): string {
  const stamp = Date.now().toString().slice(-5);
  const random = Math.floor(Math.random() * 1000)
    .toString()
    .padStart(3, '0');
  return prefix + stamp + random;
}

async function login(page: Page, phone: string, password: string) {
  await page.goto('/masuk');
  await page.getByLabel('Nomor HP').fill(phone);
  await page.getByLabel('Kata sandi', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Masuk' }).click();
}

test('daftar depot, siapkan, tambah kurir, lalu masuk sebagai kurir', async ({ page }) => {
  const ownerPhone = uniquePhone('0812');
  const courierPhone = uniquePhone('0813');

  await page.goto('/daftar');
  await page.getByLabel('Nama depot').fill('Depot Uji Otomatis');
  await page.getByLabel('Nama Anda').fill('Pemilik Uji');
  await page.getByLabel('Nomor HP').fill(ownerPhone);
  await page.getByLabel('Kata sandi', { exact: true }).fill(OWNER_PASSWORD);
  await page.getByRole('button', { name: 'Daftar' }).click();

  await expect(page).toHaveURL(/\/app\/penyiapan$/);
  await page.getByLabel('Harga isi ulang (Rp)').fill('7000');
  await page.getByLabel('Ongkos kirim (Rp)').fill('2000');
  await page.getByLabel('Alamat depot').fill('Jl. Mawar 1');
  await page.getByRole('button', { name: 'Simpan dan mulai' }).click();
  await expect(page).toHaveURL(/\/app$/);

  await page.goto('/app/pengaturan');
  await page.getByRole('tab', { name: 'Kurir' }).click();
  await page.getByLabel('Nama kurir').fill('Kurir Uji');
  await page.getByLabel('Nomor HP kurir').fill(courierPhone);
  await page.getByLabel('Kata sandi kurir').fill(COURIER_PASSWORD);
  await page.getByRole('button', { name: 'Tambah kurir' }).click();
  await expect(page.getByRole('listitem').filter({ hasText: 'Kurir Uji' })).toBeVisible();

  await page.goto('/app/lainnya');
  await page.getByRole('main').getByRole('button', { name: 'Keluar' }).click();
  await expect(page).toHaveURL(/\/masuk$/);

  await login(page, courierPhone, COURIER_PASSWORD);
  await expect(page).toHaveURL(/\/kurir$/);
  await expect(page.getByText('Antrean antar akan muncul di sini.')).toBeVisible();

  await page.goto('/app');
  await expect(page).toHaveURL(/\/kurir$/);
});

test('masuk dengan kata sandi salah menampilkan pesan galat', async ({ page }) => {
  await login(page, '081200009999', 'bukan-sandi-benar');
  await expect(page.getByRole('alert')).toContainText('Nomor atau kata sandi salah.');
});
