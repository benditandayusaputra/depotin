import {
  expect,
  test as base,
  type APIRequestContext,
  type Browser,
  type BrowserContextOptions,
  type Page
} from '@playwright/test';

const PASSWORD = 'demo-depotin-2026';
const OWNER_PHONE = '081200000001';
const COURIER_PHONE = '081200000002';
const DEPOT_SLUG = 'depot-tirta-sejuk';
const BASE_URL = 'http://localhost:5173';
const OUT_DIR = new URL('../../../docs/screenshots/', import.meta.url).pathname;
const LOGIN_ATTEMPTS = 4;
const HEADERS = { origin: BASE_URL };
const MOBILE = { viewport: { width: 360, height: 780 }, isMobile: true, hasTouch: true };
const DESKTOP = { viewport: { width: 1280, height: 800 } };

interface Envelope<T> {
  data: T;
}

interface Order {
  id: string;
  code: string;
  status: string;
}

interface Seed {
  rinaId: string;
  personalPath: string;
  trackToken: string;
  confirmedCode: string;
  deliveringCode: string;
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
  return '0858' + stamp + random;
}

async function courierId(request: APIRequestContext): Promise<string> {
  const users = await call<{ id: string; role: string; phone: string }[]>(request, 'get', '/users');
  const courier = users.find((u) => u.role === 'courier' && u.phone.endsWith('1200000002'));
  if (!courier) throw new Error('Kurir Andi tidak ada di data contoh.');
  return courier.id;
}

async function assignedOrder(
  request: APIRequestContext,
  customerId: string,
  courier: string,
  qty: number
): Promise<Order> {
  const order = await call<Order>(request, 'post', '/orders', {
    customer_id: customerId,
    refill_qty: qty
  });
  if (order.status === 'pending') await call(request, 'post', `/orders/${order.id}/confirm`);
  await call(request, 'post', `/orders/${order.id}/assign`, { courier_id: courier });
  return order;
}

async function seedReminder(request: APIRequestContext): Promise<void> {
  const courier = await courierId(request);
  const customer = await call<{ id: string }>(request, 'post', '/customers', {
    name: 'Pak Budi Santoso',
    phone: uniquePhone(),
    address: 'Jl. Melati No. 12'
  });
  const order = await assignedOrder(request, customer.id, courier, 1);
  await call(request, 'post', `/orders/${order.id}/dispatch`);
  await call(request, 'post', `/orders/${order.id}/deliver`, {
    gallons_returned: 1,
    payment_method: 'cash',
    paid: true
  });
}

const TEST_NAME = /^(Uji |Audit |Pelanggan Uji)/;

async function customerNamed(request: APIRequestContext, q: string): Promise<{ id: string }> {
  const customers = await call<{ id: string; name: string }[]>(
    request,
    'get',
    `/customers?q=${encodeURIComponent(q)}`
  );
  const found = customers.find((c) => c.name.includes(q) && !TEST_NAME.test(c.name));
  if (!found) throw new Error(`Pelanggan ${q} tidak ada di data contoh.`);
  return found;
}

async function clearStaleOrders(request: APIRequestContext): Promise<void> {
  const today = new Date().toISOString().slice(0, 10);
  const listed = (status: string) =>
    call<{ id: string; delivery_name: string }[]>(
      request,
      'get',
      `/orders?status=${status}&date=${today}&limit=100`
    );
  for (const order of await listed('on_delivery')) {
    await call(request, 'post', `/orders/${order.id}/deliver`, {
      gallons_returned: 0,
      payment_method: 'cash',
      paid: true
    });
  }
  for (const status of ['pending', 'confirmed']) {
    for (const order of await listed(status)) {
      if (!TEST_NAME.test(order.delivery_name)) continue;
      await call(request, 'post', `/orders/${order.id}/cancel`, { reason: 'data uji' });
    }
  }
}

async function seedAll(request: APIRequestContext): Promise<Seed> {
  await clearStaleOrders(request);
  const rina = await customerNamed(request, 'Rina');
  const second = await customerNamed(request, 'Joko');
  const courier = await courierId(request);

  const confirmed = await assignedOrder(request, rina.id, courier, 2);
  const delivering = await assignedOrder(request, second.id, courier, 1);
  await call(request, 'post', `/orders/${delivering.id}/dispatch`);

  const link = await call<{ link: string }>(request, 'post', `/customers/${rina.id}/link`);
  const linkUrl = new URL(link.link);

  const created = await call<{ track_token: string }>(
    request,
    'post',
    `/public/depots/${DEPOT_SLUG}/orders`,
    {
      name: 'Ibu Sari Dewi',
      phone: uniquePhone(),
      address: 'Jl. Kenanga No. 8, RT 03',
      qty: 2
    }
  );

  return {
    rinaId: rina.id,
    personalPath: linkUrl.pathname + linkUrl.search,
    trackToken: created.track_token,
    confirmedCode: confirmed.code,
    deliveringCode: delivering.code
  };
}

const test = base.extend<object, { ownerState: string; courierState: string; seed: Seed }>({
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
  seed: [
    async ({ browser, ownerState }, use) => {
      const context = await browser.newContext({ baseURL: BASE_URL, storageState: ownerState });
      try {
        await use(await seedAll(context.request));
      } finally {
        await context.close();
      }
    },
    { scope: 'worker' }
  ]
});

async function settle(page: Page): Promise<void> {
  await page.waitForLoadState('domcontentloaded');
  await expect(page.locator('[aria-busy="true"], .animate-pulse')).toHaveCount(0, {
    timeout: 20_000
  });
  if (new URL(page.url()).pathname.startsWith('/app')) {
    await expect(page.getByText('Tersambung').filter({ visible: true }).first()).toBeVisible({
      timeout: 30_000
    });
  }
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(600);
}

async function view(
  page: Page,
  viewport: { width: number; height: number },
  path: string
): Promise<void> {
  await page.setViewportSize(viewport);
  if (new URL(page.url()).pathname.startsWith('/app')) {
    await page.evaluate((href) => {
      const anchor = document.createElement('a');
      anchor.href = href;
      document.body.append(anchor);
      anchor.click();
      anchor.remove();
    }, path);
    await page.waitForURL(path);
  } else {
    await page.goto(path);
  }
  await settle(page);
}

async function open(
  browser: Browser,
  options: BrowserContextOptions,
  path: string,
  storageState?: string
): Promise<Page> {
  const context = await browser.newContext({ ...options, baseURL: BASE_URL, storageState });
  const page = await context.newPage();
  await page.goto(path);
  await settle(page);
  return page;
}

async function capture(page: Page, name: string, fullPage = false): Promise<void> {
  await page.screenshot({ path: `${OUT_DIR}${name}.png`, fullPage });
}

test.describe.configure({ mode: 'serial' });

test('halaman pemilik', async ({ browser, ownerState, seed }) => {
  test.setTimeout(300_000);
  const page = await open(browser, MOBILE, '/app/pengingat', ownerState);
  await expect(page.getByRole('heading', { level: 1, name: 'Pengingat' })).toBeVisible();
  if ((await page.getByRole('listitem').count()) === 0) {
    await call(page.request, 'patch', '/depot', { default_days_per_gallon: 0.5 });
    try {
      await seedReminder(page.request);
      await view(page, MOBILE.viewport, '/app');
      await view(page, MOBILE.viewport, '/app/pengingat');
    } finally {
      await call(page.request, 'patch', '/depot', { default_days_per_gallon: 4 });
    }
  }
  await expect(page.getByRole('listitem').first()).toBeVisible();
  await capture(page, 'pengingat-360');

  await view(page, DESKTOP.viewport, '/app');
  await capture(page, 'hari-ini-1280');

  await view(page, DESKTOP.viewport, '/app/pesanan');
  await expect(page.getByText(seed.confirmedCode)).toBeVisible();
  await capture(page, 'pesanan-1280');

  await view(page, MOBILE.viewport, '/app/pesanan/baru');
  await page.getByLabel('Cari pelanggan').fill('Rina');
  await page.getByRole('button', { name: /Bu Rina/ }).click();
  await page.getByRole('spinbutton', { name: 'Jumlah galon' }).fill('2');
  await settle(page);
  await capture(page, 'pesanan-baru-360');

  await view(page, MOBILE.viewport, `/app/pelanggan/${seed.rinaId}`);
  await expect(page.getByRole('heading', { level: 1, name: /Rina/ })).toBeVisible();
  await capture(page, 'pelanggan-detail-360', true);

  await view(page, MOBILE.viewport, '/app/galon');
  await capture(page, 'galon-360');

  await view(page, DESKTOP.viewport, '/app/laporan');
  await expect(page.getByRole('heading', { level: 1, name: 'Laporan' })).toBeVisible();
  await capture(page, 'laporan-1280', true);
  await page.context().close();
});

test('halaman kurir', async ({ browser, courierState, seed }) => {
  const page = await open(browser, MOBILE, '/kurir', courierState);
  const confirmed = page.getByRole('listitem').filter({ hasText: seed.confirmedCode });
  const delivering = page.getByRole('listitem').filter({ hasText: seed.deliveringCode });
  await expect(confirmed).toBeVisible({ timeout: 20_000 });
  await expect(delivering).toBeVisible();
  await capture(page, 'kurir-360');
  await delivering.getByRole('button', { name: 'Selesai', exact: true }).click();
  const sheet = page.getByRole('dialog');
  await expect(sheet).toBeVisible();
  await sheet.getByRole('button', { name: 'Tunai' }).click();
  await page.waitForTimeout(600);
  await capture(page, 'kurir-selesai-360');
  await page.context().close();
});

test('halaman publik', async ({ browser, seed }) => {
  let page = await open(browser, MOBILE, seed.personalPath);
  await expect(page.getByRole('heading', { level: 1, name: /^Halo, / })).toBeVisible();
  await capture(page, 'pribadi-360');
  await page.context().close();

  page = await open(browser, MOBILE, `/d/${DEPOT_SLUG}`);
  await expect(page.getByRole('heading', { level: 1, name: 'Depot Tirta Sejuk' })).toBeVisible();
  await capture(page, 'depot-publik-360');
  await page.context().close();

  page = await open(browser, MOBILE, `/t/${seed.trackToken}`);
  await expect(page.getByText('Menunggu konfirmasi')).toBeVisible();
  await capture(page, 'lacak-360');
  await page.context().close();
});
