<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { apiFetch, newIdempotencyKey, type ApiFetchInit } from '$lib/api/client';
  import type { Order } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Chip from '$lib/components/Chip.svelte';
  import QtyStepper from '$lib/components/QtyStepper.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SelectField from '$lib/components/SelectField.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import Toggle from '$lib/components/Toggle.svelte';
  import type { User } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { stream } from '$lib/stream/client.svelte';
  import { formatDate, formatDateTime } from '$lib/utils/date';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';
  import {
    PAYMENT_METHODS,
    fulfilmentLabel,
    orderSourceLabel,
    paymentMethodLabel,
    type PaymentMethod
  } from '$lib/utils/labels';
  import { formatRupiah } from '$lib/utils/rupiah';

  type Panel = 'none' | 'assign' | 'deliver' | 'cancel' | 'pay';

  const orderId = $derived(page.params.id ?? '');

  let order = $state<Order | null>(null);
  let couriers = $state<User[]>([]);
  let loading = $state(true);
  let loadError = $state('');

  let panel = $state<Panel>('none');
  let busy = $state('');
  let actionError = $state('');
  let errors = $state<Record<string, string>>({});

  let courierId = $state('');
  let gallonsReturned = $state(0);
  let paymentMethod = $state<PaymentMethod>('cash');
  let paid = $state(true);
  let cancelReason = $state('');

  const timeline = $derived.by(() => {
    if (!order) return [];
    const steps: { label: string; at: string | null }[] = [
      { label: 'Dibuat', at: order.created_at },
      { label: 'Dikonfirmasi', at: order.confirmed_at },
      { label: 'Berangkat', at: order.dispatched_at },
      { label: 'Selesai', at: order.delivered_at },
      { label: 'Dibatalkan', at: order.cancelled_at }
    ];
    return steps.filter((step) => step.at !== null);
  });

  const activeCouriers = $derived(couriers.filter((courier) => courier.is_active));
  const canCancel = $derived(
    order !== null && order.status !== 'delivered' && order.status !== 'cancelled'
  );

  async function load(silent = false) {
    if (!silent) loading = true;
    loadError = '';
    try {
      order = await apiFetch<Order>(`/orders/${orderId}`);
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  async function loadCouriers() {
    try {
      const users = await apiFetch<User[]>('/users');
      couriers = users.filter((user) => user.role === 'courier');
    } catch {
      couriers = [];
    }
  }

  function openPanel(next: Panel) {
    panel = panel === next ? 'none' : next;
    actionError = '';
    errors = {};
    if (next === 'deliver' && order) gallonsReturned = order.refill_qty;
    if (next === 'assign' && order) courierId = order.courier_id ?? activeCouriers[0]?.id ?? '';
  }

  async function act(name: string, path: string, init: ApiFetchInit, successMessage: string) {
    busy = name;
    actionError = '';
    errors = {};
    try {
      order = await apiFetch<Order>(`/orders/${orderId}/${path}`, { method: 'POST', ...init });
      panel = 'none';
      pushToast('success', successMessage);
      return true;
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) actionError = errorMessage(error);
      return false;
    } finally {
      busy = '';
    }
  }

  async function confirm() {
    if (!order) return;
    const previous = order;
    order = { ...order, status: 'confirmed', confirmed_at: new Date().toISOString() };
    const ok = await act('confirm', 'confirm', {}, 'Pesanan dikonfirmasi.');
    if (!ok) order = previous;
  }

  function assign(event: SubmitEvent) {
    event.preventDefault();
    void act('assign', 'assign', { body: { courier_id: courierId } }, 'Kurir ditugaskan.');
  }

  function dispatch() {
    void act(
      'dispatch',
      'dispatch',
      { idempotencyKey: newIdempotencyKey() },
      'Pesanan sedang diantar.'
    );
  }

  function deliver(event: SubmitEvent) {
    event.preventDefault();
    void act(
      'deliver',
      'deliver',
      {
        body: { gallons_returned: gallonsReturned, payment_method: paymentMethod, paid },
        idempotencyKey: newIdempotencyKey()
      },
      'Pesanan selesai.'
    );
  }

  function markPaid(event: SubmitEvent) {
    event.preventDefault();
    void act('pay', 'mark-paid', { body: { payment_method: paymentMethod } }, 'Pesanan lunas.');
  }

  function cancel(event: SubmitEvent) {
    event.preventDefault();
    void act('cancel', 'cancel', { body: { reason: cancelReason } }, 'Pesanan dibatalkan.');
  }

  onMount(() => {
    void load();
    void loadCouriers();
    return stream.subscribe((event) => {
      if (event.type === 'poll' || (event.type === 'order.updated' && event.data.id === orderId)) {
        void load(true);
      }
    });
  });
</script>

<svelte:head>
  <title>{order ? order.code : 'Pesanan'} | Depotin</title>
</svelte:head>

<a
  href={resolve('/app/pesanan')}
  class="mb-2 inline-flex min-h-11 items-center font-semibold text-accent-700 dark:text-accent-300"
>
  &larr; Semua pesanan
</a>

{#if loading}
  <Skeleton lines={10} />
{:else if loadError || !order}
  <Alert kind="error" onretry={() => void load()}>{loadError}</Alert>
{:else}
  <PageHeader title={order.code} description={orderSourceLabel(order.source)}>
    {#snippet actions()}
      {#if order}
        <StatusBadge status={order.status} />
      {/if}
    {/snippet}
  </PageHeader>

  <div class="flex flex-col gap-4">
    <Card>
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <a
            href={resolve('/app/pelanggan/[id]', { id: order.customer_id })}
            class="inline-flex min-h-11 items-center text-xl font-bold text-accent-700 underline dark:text-accent-300"
          >
            {order.delivery_name}
          </a>
          <p class="text-muted">{order.delivery_phone}</p>
          <p>{order.delivery_address}</p>
          {#if order.delivery_note}
            <p class="text-muted">{order.delivery_note}</p>
          {/if}
        </div>
        <div class="text-right">
          <p class="text-sm text-muted">Total</p>
          <p class="text-2xl font-bold" aria-label="Total pesanan">{formatRupiah(order.total)}</p>
          <Badge tone={order.payment_status === 'paid' ? 'done' : 'pending'}>
            {order.payment_status === 'paid' ? 'Lunas' : 'Belum bayar'}
          </Badge>
        </div>
      </div>
      <dl class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
        <div>
          <dt class="text-muted">Cara ambil</dt>
          <dd class="font-medium">{fulfilmentLabel(order.fulfilment)}</dd>
        </div>
        <div>
          <dt class="text-muted">Hari antar</dt>
          <dd class="font-medium">{formatDate(`${order.scheduled_date}T12:00:00+07:00`)}</dd>
        </div>
        <div>
          <dt class="text-muted">Kurir</dt>
          <dd class="font-medium">{order.courier_name ?? 'Belum ada'}</dd>
        </div>
        <div>
          <dt class="text-muted">Galon isi ulang</dt>
          <dd class="font-medium">{order.refill_qty}</dd>
        </div>
        {#if order.gallons_returned !== null}
          <div>
            <dt class="text-muted">Galon kosong diterima</dt>
            <dd class="font-medium">{order.gallons_returned}</dd>
          </div>
        {/if}
        {#if order.payment_method}
          <div>
            <dt class="text-muted">Cara bayar</dt>
            <dd class="font-medium">{paymentMethodLabel(order.payment_method)}</dd>
          </div>
        {/if}
        {#if order.note}
          <div class="col-span-full">
            <dt class="text-muted">Catatan</dt>
            <dd class="font-medium">{order.note}</dd>
          </div>
        {/if}
        {#if order.cancel_reason}
          <div class="col-span-full">
            <dt class="text-muted">Alasan batal</dt>
            <dd class="font-medium">{order.cancel_reason}</dd>
          </div>
        {/if}
      </dl>
    </Card>

    <Card>
      <h2 class="mb-3 text-lg font-semibold">Rincian tagihan</h2>
      <ul class="divide-y divide-border">
        {#each order.items as item (item.product_id)}
          <li class="flex justify-between gap-3 py-2">
            <span>{item.product_name} &times; {item.qty}</span>
            <span class="font-medium">{formatRupiah(item.line_total)}</span>
          </li>
        {/each}
        {#if order.delivery_fee > 0}
          <li class="flex justify-between gap-3 py-2">
            <span>Ongkos kirim</span>
            <span class="font-medium">{formatRupiah(order.delivery_fee)}</span>
          </li>
        {/if}
        {#if order.discount > 0}
          <li class="flex justify-between gap-3 py-2 text-status-done">
            <span>Potongan loyalitas ({order.free_qty} galon gratis)</span>
            <span class="font-medium">-{formatRupiah(order.discount)}</span>
          </li>
        {/if}
        <li class="flex justify-between gap-3 py-2 text-lg font-bold">
          <span>Total</span>
          <span>{formatRupiah(order.total)}</span>
        </li>
      </ul>
    </Card>

    <Card>
      <h2 class="mb-3 text-lg font-semibold">Riwayat</h2>
      <ol class="flex flex-col gap-2">
        {#each timeline as step (step.label)}
          <li class="flex items-center justify-between gap-3">
            <span class="font-medium">{step.label}</span>
            <span class="text-sm text-muted">{formatDateTime(step.at ?? '')}</span>
          </li>
        {/each}
      </ol>
    </Card>

    {#if order.status !== 'cancelled' && (order.status !== 'delivered' || order.payment_status === 'unpaid')}
      <Card>
        <h2 class="mb-3 text-lg font-semibold">Tindakan</h2>
        {#if actionError}
          <div class="mb-3"><Alert kind="error">{actionError}</Alert></div>
        {/if}
        <div class="flex flex-wrap gap-2">
          {#if order.status === 'pending'}
            <Button size="lg" loading={busy === 'confirm'} onclick={() => void confirm()}>
              Konfirmasi
            </Button>
          {/if}
          {#if order.status === 'confirmed'}
            <Button size="lg" variant="secondary" onclick={() => openPanel('assign')}>
              Tugaskan kurir
            </Button>
            {#if order.fulfilment === 'delivery'}
              <Button
                size="lg"
                loading={busy === 'dispatch'}
                disabled={!order.courier_id}
                onclick={dispatch}
              >
                Berangkat
              </Button>
            {/if}
          {/if}
          {#if order.status === 'confirmed' || order.status === 'on_delivery'}
            <Button size="lg" onclick={() => openPanel('deliver')}>Selesai</Button>
          {/if}
          {#if order.status === 'delivered' && order.payment_status === 'unpaid'}
            <Button size="lg" onclick={() => openPanel('pay')}>Tandai lunas</Button>
          {/if}
          {#if canCancel}
            <Button size="lg" variant="danger" onclick={() => openPanel('cancel')}>Batalkan</Button>
          {/if}
        </div>
        {#if order.status === 'confirmed' && order.fulfilment === 'delivery' && !order.courier_id}
          <p class="mt-2 text-sm text-muted">Tugaskan kurir dulu sebelum berangkat.</p>
        {/if}

        {#if panel === 'assign'}
          <form class="mt-4 flex flex-col gap-3" onsubmit={assign}>
            <SelectField label="Kurir" bind:value={courierId} error={errors.courier_id}>
              <option value="" disabled>Pilih kurir</option>
              {#each activeCouriers as courier (courier.id)}
                <option value={courier.id}>{courier.name}</option>
              {/each}
            </SelectField>
            {#if activeCouriers.length === 0}
              <p class="text-muted">Belum ada kurir aktif. Tambahkan di Pengaturan.</p>
            {/if}
            <div class="flex gap-2">
              <Button type="submit" loading={busy === 'assign'} disabled={!courierId}>
                Simpan kurir
              </Button>
              <Button variant="ghost" onclick={() => openPanel('none')}>Batal</Button>
            </div>
          </form>
        {:else if panel === 'deliver'}
          <form class="mt-4 flex flex-col gap-4" onsubmit={deliver}>
            <QtyStepper
              label="Galon kosong diterima"
              bind:value={gallonsReturned}
              error={errors.gallons_returned}
            />
            <fieldset>
              <legend class="mb-2 font-medium">Cara bayar</legend>
              <div class="flex flex-wrap gap-2">
                {#each PAYMENT_METHODS as method (method)}
                  <Chip
                    selected={paymentMethod === method}
                    onclick={() => (paymentMethod = method)}
                  >
                    {paymentMethodLabel(method)}
                  </Chip>
                {/each}
              </div>
            </fieldset>
            <Toggle label="Sudah dibayar" bind:checked={paid} />
            <div class="flex gap-2">
              <Button type="submit" size="lg" loading={busy === 'deliver'}>Simpan selesai</Button>
              <Button variant="ghost" onclick={() => openPanel('none')}>Batal</Button>
            </div>
          </form>
        {:else if panel === 'pay'}
          <form class="mt-4 flex flex-col gap-4" onsubmit={markPaid}>
            <fieldset>
              <legend class="mb-2 font-medium">Cara bayar</legend>
              <div class="flex flex-wrap gap-2">
                {#each PAYMENT_METHODS as method (method)}
                  <Chip
                    selected={paymentMethod === method}
                    onclick={() => (paymentMethod = method)}
                  >
                    {paymentMethodLabel(method)}
                  </Chip>
                {/each}
              </div>
            </fieldset>
            <div class="flex gap-2">
              <Button type="submit" size="lg" loading={busy === 'pay'}>Simpan lunas</Button>
              <Button variant="ghost" onclick={() => openPanel('none')}>Batal</Button>
            </div>
          </form>
        {:else if panel === 'cancel'}
          <form class="mt-4 flex flex-col gap-3" onsubmit={cancel}>
            <Alert kind="error">Pesanan yang dibatalkan tidak bisa dikembalikan.</Alert>
            <TextField
              label="Alasan pembatalan"
              bind:value={cancelReason}
              error={errors.reason}
              placeholder="Minimal 3 huruf"
              required
            />
            <div class="flex gap-2">
              <Button
                type="submit"
                variant="danger"
                size="lg"
                loading={busy === 'cancel'}
                disabled={cancelReason.trim().length < 3}
              >
                Ya, batalkan pesanan
              </Button>
              <Button variant="ghost" onclick={() => openPanel('none')}>Jangan</Button>
            </div>
          </form>
        {/if}
      </Card>
    {/if}
  </div>
{/if}
