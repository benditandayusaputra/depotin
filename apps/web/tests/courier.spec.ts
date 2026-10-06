import { expect, test, type APIRequestContext } from '@playwright/test';

const PASSWORD = 'demo-depotin-2026';
const OWNER_PHONE = '081200000001';
const COURIER_PHONE = '081200000002';

interface Envelope<T> {
  data: T;
}

async function readData<T>(
  promise: Promise<{ ok(): boolean; json(): Promise<unknown> }>
): Promise<T> {
  const response = await promise;
  expect(response.ok()).toBeTruthy();
  return ((await response.json()) as Envelope<T>).data;
}

async function createAssignedOrder(request: APIRequestContext, origin: string): Promise<string> {
  const headers = { origin };
  await readData(
    request.post('/api/v1/auth/login', {
      headers,
      data: { phone: OWNER_PHONE, password: PASSWORD }
    })
  );
  const customers = await readData<{ id: string }[]>(
    request.get('/api/v1/customers?q=Rina', { headers })
  );
  const customer = customers[0];
  if (!customer) throw new Error('Pelanggan Rina tidak ada di data contoh.');

  const users = await readData<{ id: string; role: string; phone: string }[]>(
    request.get('/api/v1/users', { headers })
  );
  const courier = users.find(
    (user) => user.role === 'courier' && user.phone.endsWith('1200000002')
  );
  if (!courier) throw new Error('Kurir demo tidak ada di data contoh.');

  const order = await readData<{ id: string; code: string }>(
    request.post('/api/v1/orders', {
      headers: { ...headers, 'idempotency-key': crypto.randomUUID() },
      data: { customer_id: customer.id, refill_qty: 2 }
    })
  );
  await readData(
    request.post(`/api/v1/orders/${order.id}/assign`, {
      headers,
      data: { courier_id: courier.id }
    })
  );
  await request.post('/api/v1/auth/logout', { headers });
  return order.code;
}

test('kurir berangkat lalu menyelesaikan pesanan dengan tunai', async ({ page, baseURL }) => {
  const code = await createAssignedOrder(page.request, baseURL ?? '');

  await page.goto('/masuk');
  await page.getByLabel('Nomor HP').fill(COURIER_PHONE);
  await page.getByLabel('Kata sandi', { exact: true }).fill(PASSWORD);
  await page.getByRole('button', { name: 'Masuk' }).click();
  await expect(page).toHaveURL(/\/kurir$/);

  const card = page.getByRole('listitem').filter({ hasText: code });
  await expect(card).toBeVisible();
  await expect(card.getByText('Dikonfirmasi')).toBeVisible();

  await card.getByRole('button', { name: 'Berangkat' }).click();
  await expect(card.getByText('Sedang diantar')).toBeVisible();

  await card.getByRole('button', { name: 'Selesai', exact: true }).click();
  const sheet = page.getByRole('dialog');
  await expect(sheet.getByRole('button', { name: 'Selesai', exact: true })).toBeDisabled();
  await sheet.getByRole('button', { name: 'Tunai' }).click();
  await sheet.getByRole('button', { name: 'Selesai', exact: true }).click();

  await expect(page.getByText(`Pesanan ${code} selesai.`)).toBeVisible();
  await expect(card).toHaveCount(0);
});
