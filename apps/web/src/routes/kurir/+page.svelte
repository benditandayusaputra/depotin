<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError, apiFetch, newIdempotencyKey } from '$lib/api/client';
  import type { components } from '$lib/api/schema';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import DeliverSheet, { type DeliverPayload } from '$lib/components/courier/DeliverSheet.svelte';
  import {
    indexedDbStore,
    OfflineQueue,
    type QueuedAction,
    type QueuedActionKind
  } from '$lib/offline/queue';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage } from '$lib/utils/errors';
  import { pollWhileVisible } from '$lib/utils/poll';
  import { formatRupiah } from '$lib/utils/rupiah';
  import { waUrl } from '$lib/utils/whatsapp';

  type CourierOrder = components['schemas']['CourierOrder'];

  const POLL_MS = 30_000;
  const QUEUED_MESSAGE = 'Tersimpan, akan dikirim saat sinyal kembali';
  const LINK_BUTTON =
    'inline-flex min-h-12 items-center justify-center rounded-lg border border-border bg-surface px-4 font-semibold text-accent-700 dark:text-accent-300';

  let orders = $state<CourierOrder[]>([]);
  let loading = $state(true);
  let loadError = $state('');
  let pending = $state(0);
  let finishing = $state<CourierOrder | null>(null);

  async function sendAction(action: QueuedAction): Promise<number> {
    try {
      await apiFetch(`/orders/${action.orderId}/${action.action}`, {
        method: 'POST',
        body: action.body,
        idempotencyKey: action.idempotencyKey
      });
      return 200;
    } catch (error) {
      if (error instanceof ApiError) return error.status;
      throw error;
    }
  }

  const queue = new OfflineQueue(indexedDbStore(), sendAction);

  async function flush() {
    pending = await queue.flush();
  }

  async function loadQueue() {
    loadError = '';
    try {
      orders = await apiFetch<CourierOrder[]>('/courier/queue');
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  function byDeliveryFirst(list: CourierOrder[]): CourierOrder[] {
    return [...list].sort(
      (a, b) => Number(b.status === 'on_delivery') - Number(a.status === 'on_delivery')
    );
  }

  function mapUrl(order: CourierOrder): string {
    const query =
      order.lat !== null && order.lng !== null
        ? `${order.lat},${order.lng}`
        : encodeURIComponent(order.delivery_address);
    return `https://www.google.com/maps/search/?api=1&query=${query}`;
  }

  async function enqueue(action: Omit<QueuedAction, 'id' | 'createdAt'>) {
    await queue.enqueue(action);
    pending = await queue.count();
    pushToast('info', QUEUED_MESSAGE);
  }

  async function perform(
    order: CourierOrder,
    action: QueuedActionKind,
    body: Record<string, unknown> | undefined,
    apply: () => void,
    revert: () => void
  ): Promise<boolean> {
    apply();
    const item = { orderId: order.id, action, body, idempotencyKey: newIdempotencyKey() };
    if (!navigator.onLine) {
      await enqueue(item);
      return false;
    }
    try {
      await apiFetch(`/orders/${order.id}/${action}`, {
        method: 'POST',
        body,
        idempotencyKey: item.idempotencyKey
      });
      return true;
    } catch (error) {
      if (error instanceof TypeError) {
        await enqueue(item);
        return false;
      }
      revert();
      pushToast('error', errorMessage(error));
      void loadQueue();
      return false;
    }
  }

  function dispatch(order: CourierOrder) {
    const previous = orders;
    void perform(
      order,
      'dispatch',
      undefined,
      () => {
        orders = byDeliveryFirst(
          orders.map((item) =>
            item.id === order.id ? { ...item, status: 'on_delivery' as const } : item
          )
        );
      },
      () => {
        orders = previous;
      }
    );
  }

  function saveLocation(order: CourierOrder) {
    const failed = () => pushToast('info', 'Lokasi belum tersimpan.');
    if (!('geolocation' in navigator)) {
      failed();
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (position) => {
        void apiFetch(`/courier/customers/${order.customer_id}/location`, {
          body: { lat: position.coords.latitude, lng: position.coords.longitude }
        }).then(() => pushToast('success', 'Lokasi tersimpan.'), failed);
      },
      failed,
      { timeout: 10_000 }
    );
  }

  async function deliver(payload: DeliverPayload, withLocation: boolean) {
    const order = finishing;
    if (!order) return;
    finishing = null;
    const previous = orders;
    const sent = await perform(
      order,
      'deliver',
      { ...payload },
      () => {
        orders = orders.filter((item) => item.id !== order.id);
      },
      () => {
        orders = previous;
      }
    );
    if (!sent) return;
    pushToast('success', `Pesanan ${order.code} selesai.`);
    if (withLocation) saveLocation(order);
  }

  $effect(() => pollWhileVisible(POLL_MS, () => void loadQueue()));

  onMount(() => {
    void loadQueue();
    void flush();
    const onOnline = () => void flush().then(loadQueue);
    window.addEventListener('online', onOnline);
    return () => window.removeEventListener('online', onOnline);
  });
</script>

<svelte:head>
  <title>Antar | Depotin</title>
</svelte:head>

<PageHeader title="Antrean antar">
  {#snippet actions()}
    <Button variant="secondary" size="lg" onclick={() => void loadQueue()}>Muat ulang</Button>
  {/snippet}
</PageHeader>

{#if pending > 0}
  <div class="mb-4">
    <Alert kind="info">{pending} aksi menunggu sinyal untuk dikirim.</Alert>
  </div>
{/if}

{#if loading}
  <div aria-busy="true"><Skeleton lines={8} /></div>
{:else}
  {#if loadError}
    <div class="mb-4">
      <Alert kind="error" onretry={() => void loadQueue()}>{loadError}</Alert>
    </div>
  {/if}
  {#if orders.length === 0 && !loadError}
    <EmptyState
      title="Antrean antar akan muncul di sini."
      description="Tekan Muat ulang bila pemilik baru saja menugaskan pesanan."
    />
  {:else}
    <ul class="flex flex-col gap-4">
      {#each orders as order (order.id)}
        <li>
          <Card class="flex flex-col gap-3">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <p class="text-xl font-bold">{order.delivery_name}</p>
                <p class="text-muted">{order.code}</p>
              </div>
              <StatusBadge status={order.status} />
            </div>
            <p>{order.delivery_address}</p>
            {#if order.delivery_note}
              <p class="text-muted">Patokan: {order.delivery_note}</p>
            {/if}
            {#if order.area}
              <p class="text-muted">Area {order.area}</p>
            {/if}
            {#if order.note}
              <p class="text-muted">Catatan: {order.note}</p>
            {/if}
            <div class="flex items-end justify-between gap-3">
              <p class="text-2xl font-bold">
                {order.refill_qty} galon
                {#if order.free_qty > 0}
                  <span class="text-base font-medium text-status-done">+{order.free_qty} gratis</span>
                {/if}
              </p>
              <p class="text-right">
                {#if order.payment_status === 'paid'}
                  <span class="font-semibold text-status-done">Lunas</span>
                {:else}
                  <span class="text-muted">Tagih</span>
                  <span class="block text-xl font-bold tabular-nums">{formatRupiah(order.total)}</span>
                {/if}
              </p>
            </div>
            <div class="grid grid-cols-2 gap-2">
              <a href={mapUrl(order)} target="_blank" rel="external noopener" class={LINK_BUTTON}>Peta</a>
              <a
                href={waUrl(order.delivery_phone, `Halo ${order.delivery_name}, galon segera diantar.`)}
                target="_blank"
                rel="external noopener"
                class={LINK_BUTTON}
              >
                WhatsApp
              </a>
            </div>
            {#if order.status === 'confirmed'}
              <Button size="lg" onclick={() => dispatch(order)}>Berangkat</Button>
            {:else}
              <Button size="lg" onclick={() => (finishing = order)}>Selesai</Button>
            {/if}
          </Card>
        </li>
      {/each}
    </ul>
  {/if}
{/if}

{#if finishing}
  <DeliverSheet order={finishing} onsubmit={deliver} onclose={() => (finishing = null)} />
{/if}
