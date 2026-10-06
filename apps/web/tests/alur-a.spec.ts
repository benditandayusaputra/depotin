import {
  expect,
  test as base,
  type APIRequestContext,
  type Browser,
  type Page
} from '@playwright/test';

const PASSWORD = 'demo-depotin-2026';
const OWNER_PHONE = '081200000001';
const COURIER_PHONE = '081200000002';
const BASE_URL = 'http://localhost:5173';
const LOGIN_ATTEMPTS = 4;
const HEADERS = { origin: BASE_URL };

interface Envelope<T> {
  data: T;
}

async function loginState(browser: Browser, phone: string, name: string): Promise<string> {
  const context = await browser.newContext({ baseURL: BASE_URL });
  try {
    for (let attempt = 1; attempt <= LOGIN_ATTEMPTS; attempt += 1) {
      const response = await context.request.post('/api/v1/auth/login', {
        data: { phone, password: PASSWORD },
        headers: HEADERS
      });
      if (response.ok()) {
        const file = test.info().outputPath(`${name}-state.json`);
        await context.storageState({ path: file });
        return file;
      }
      if (response.status() !== 429 || attempt === LOGIN_ATTEMPTS) {
        throw new Error(`Masuk ${name} gagal: ${response.status()} ${await response.text()}`);
      }
      const retryAfter = Number(response.headers()['retry-after'] ?? '15');
      await new Promise((resolve) => setTimeout(resolve, (retryAfter + 1) * 1000));
    }
  } finally {
    await context.close();
  }
  throw new Error(`Masuk ${name} gagal.`);
}

const test = base.extend<object, { ownerState: string; courierState: string }>({
  ownerState: [
    async ({ browser }, use) => {
      await use(await loginState(browser, OWNER_PHONE, 'owner'));
    },
    { scope: 'worker' }
  ],
  courierState: [
    async ({ browser }, use) => {
      await use(await loginState(browser, COURIER_PHONE, 'courier'));
    },
    { scope: 'worker' }
  ],
  storageState: ({ ownerState }, use) => use(ownerState)
});

async function call<T>(
  request: APIRequestContext,
  method: 'get' | 'post' | 'patch',
  path: string,
  data?: Record<string, unknown>
): Promise<T> {
  const response = await request.fetch(`/api/v1${path}`, {
    method,
    data,
    headers: { ...HEADERS, 'idempotency-key': crypto.randomUUID() }
  });
  if (!response.ok()) {
    throw new Error(
      `${method.toUpperCase()} ${path}: ${response.status()} ${await response.text()}`
    );
  }
  return ((await response.json()) as Envelope<T>).data;
}

function uniquePhone(): string {
  const stamp = Date.now().toString().slice(-7);
  const random = Math.floor(Math.random() * 100)
    .toString()
    .padStart(2, '0');
  return '0859' + stamp + random;
}

interface Order {
  id: string;
  code: string;
  status: string;
}

async function seedReminderCondition(
  request: APIRequestContext
): Promise<{ customerId: string; customerName: string }> {
  await call(request, 'patch', '/depot', { default_days_per_gallon: 0.5 });
  const phone = uniquePhone();
  const customerName = `Uji Alur A ${phone.slice(-6)}`;
  const customer = await call<{ id: string }>(request, 'post', '/customers', {
    name: customerName,
    phone,
    address: 'Jl. Alur A No. 1'
  });
  const users = await call<{ id: string; role: string; phone: string }[]>(request, 'get', '/users');
  const courier = users.find((u) => u.role === 'courier' && u.phone.endsWith('1200000002'));
  if (!courier) throw new Error('Kurir Andi tidak ada di data contoh.');

  const order = await call<Order>(request, 'post', '/orders', {
    customer_id: customer.id,
    refill_qty: 1
  });
  if (order.status === 'pending') await call(request, 'post', `/orders/${order.id}/confirm`);
  await call(request, 'post', `/orders/${order.id}/assign`, { courier_id: courier.id });
  await call(request, 'post', `/orders/${order.id}/dispatch`);
  await call(request, 'post', `/orders/${order.id}/deliver`, {
    gallons_returned: 1,
    payment_method: 'cash',
    paid: true
  });
  return { customerId: customer.id, customerName };
}

async function captureWindowOpen(page: Page): Promise<() => Promise<string>> {
  let resolveUrl: (url: string) => void = () => undefined;
  const captured = new Promise<string>((resolve) => {
    resolveUrl = resolve;
  });
  await page.exposeFunction('captureWa', (url: string) => resolveUrl(url));
  await page.addInitScript(() => {
    window.open = (url?: string | URL) => {
      void (window as unknown as { captureWa: (u: string) => void }).captureWa(String(url));
      return null;
    };
  });
  return () => captured;
}

function personalPathFromWa(waUrl: string): string {
  const text = new URL(waUrl).searchParams.get('text') ?? '';
  const match = /https?:\/\/[^\s/]+(\/p\/[A-Za-z0-9]+\?r=[0-9a-f-]{36})/.exec(text);
  if (!match?.[1]) throw new Error(`Link pribadi tidak ditemukan di pesan WA: ${text}`);
  return match[1];
}

test.describe.configure({ mode: 'serial' });

test('alur A: pengingat, pesan dari link pribadi, antar oleh kurir, konversi tercatat', async ({
  page,
  browser,
  courierState
}) => {
  test.setTimeout(180_000);
  const request = page.request;
  try {
    const { customerId, customerName } = await seedReminderCondition(request);

    const readWaUrl = await captureWindowOpen(page);
    await page.goto('/app/pengingat');
    await expect(page.getByRole('heading', { level: 1, name: 'Pengingat' })).toBeVisible();
    const row = page.getByRole('listitem').filter({ hasText: customerName });
    await expect(row).toBeVisible({ timeout: 20_000 });
    await expect(row.getByText('Menunggu')).toBeVisible();

    await row.getByRole('button', { name: 'Kirim WA' }).click();
    const waUrl = await readWaUrl();
    expect(waUrl).toContain('wa.me');
    await expect(row.getByText('Terkirim')).toBeVisible();
    await expect(row.getByRole('button', { name: 'Kirim WA' })).toHaveCount(0);
    const personalPath = personalPathFromWa(waUrl);

    const customerContext = await browser.newContext({ baseURL: BASE_URL });
    let code = '';
    try {
      const customerPage = await customerContext.newPage();
      await customerPage.goto(personalPath);
      await expect(
        customerPage.getByRole('heading', { level: 1, name: `Halo, ${customerName}` })
      ).toBeVisible();
      await customerPage.getByRole('button', { name: /^Pesan \d+ galon seperti biasa$/ }).click();
      await expect(customerPage.getByRole('heading', { name: 'Pesanan diterima' })).toBeVisible();
      const codeText = await customerPage.getByText(/^Kode /).textContent();
      const codeMatch = /DP-[A-Z0-9-]+/.exec(codeText ?? '');
      if (!codeMatch) throw new Error(`Kode pesanan tidak terbaca: ${codeText}`);
      code = codeMatch[0];
    } finally {
      await customerContext.close();
    }

    await page.goto('/app/pesanan');
    await expect(page.getByRole('heading', { level: 1, name: 'Pesanan' })).toBeVisible();
    await page.getByRole('link', { name: `Buka pesanan ${code}` }).click();
    await expect(page.getByRole('heading', { level: 1, name: code })).toBeVisible();
    await expect(
      page.locator('header').filter({ hasText: code }).getByText('Pengingat')
    ).toBeVisible();
    const confirmButton = page.getByRole('button', { name: 'Konfirmasi' });
    if (await confirmButton.isVisible()) await confirmButton.click();
    await page.getByRole('button', { name: 'Tugaskan kurir' }).click();
    await page.getByLabel('Kurir').selectOption({ label: 'Andi' });
    await page.getByRole('button', { name: 'Simpan kurir' }).click();
    await expect(page.getByRole('definition').filter({ hasText: 'Andi' })).toBeVisible();

    const courierContext = await browser.newContext({
      baseURL: BASE_URL,
      storageState: courierState
    });
    try {
      const courierPage = await courierContext.newPage();
      await courierPage.goto('/kurir');
      const card = courierPage.getByRole('listitem').filter({ hasText: code });
      await expect(card).toBeVisible({ timeout: 20_000 });
      await expect(card.getByText(customerName)).toBeVisible();
      await card.getByRole('button', { name: 'Berangkat' }).click();
      await expect(card.getByText('Sedang diantar')).toBeVisible();
      await card.getByRole('button', { name: 'Selesai', exact: true }).click();
      const sheet = courierPage.getByRole('dialog');
      await sheet.getByRole('button', { name: 'Tunai' }).click();
      await sheet.getByRole('button', { name: 'Selesai', exact: true }).click();
      await expect(courierPage.getByText(`Pesanan ${code} selesai.`)).toBeVisible();
      await expect(card).toHaveCount(0);
    } finally {
      await courierContext.close();
    }

    await page.goto('/app/laporan');
    await expect(page.getByRole('heading', { level: 1, name: 'Laporan' })).toBeVisible();
    const conversion = page.getByText(/^\d+ dari \d+ terkirim jadi pesanan$/);
    await expect(conversion).toBeVisible();
    const converted = Number(/^(\d+) dari/.exec((await conversion.textContent()) ?? '')?.[1]);
    expect(converted).toBeGreaterThanOrEqual(1);

    await page.goto(`/app/pelanggan/${customerId}`);
    await expect(page.getByRole('heading', { level: 1, name: customerName })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Perkiraan galon habis' })).toBeVisible();
    await expect(page.getByText(/^Biasanya 1 galon habis dalam/)).toBeVisible();
    await expect(page.getByText('Belum ada prediksi')).toHaveCount(0);
    await expect(page.getByText('Galon dipinjam')).toBeVisible();
  } finally {
    await call(request, 'patch', '/depot', { default_days_per_gallon: 4 });
  }
});
