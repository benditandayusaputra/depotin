<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import type { components } from '$lib/api/schema';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { formatDate, formatTime } from '$lib/utils/date';
  import { errorMessage } from '$lib/utils/errors';
  import { pollWhileVisible } from '$lib/utils/poll';
  import { formatRupiah } from '$lib/utils/rupiah';
  import { scheduledDayLabel } from '$lib/utils/schedule';
  import { isFinalStatus, paymentLabel } from '$lib/utils/status';
  import { waUrl } from '$lib/utils/whatsapp';
  import type { PageProps } from './$types';

  type TrackedOrder = components['schemas']['TrackedOrder'];

  const POLL_MS = 15_000;

  let { data }: PageProps = $props();

  let order = $derived(data.order);
  let confirming = $state(false);
  let cancelling = $state(false);
  let hydrated = $state(false);
  let message = $state('');

  const path = $derived(`/public/track/${encodeURIComponent(data.token)}`);
  const finished = $derived(isFinalStatus(order.status));
  const steps = $derived(
    order.status === 'cancelled'
      ? [
          { label: 'Pesanan dibuat', at: order.created_at },
          { label: 'Dibatalkan', at: order.cancelled_at }
        ]
      : [
          { label: 'Pesanan dibuat', at: order.created_at },
          { label: 'Dikonfirmasi', at: order.confirmed_at },
          { label: 'Sedang diantar', at: order.dispatched_at },
          { label: 'Selesai', at: order.delivered_at }
        ]
  );
  const waLink = $derived(
    waUrl(order.depot.phone, `Halo ${order.depot.name}, saya mau tanya pesanan ${order.code}.`)
  );

  function refresh() {
    void apiFetch<TrackedOrder>(path).then(
      (fresh) => {
        order = fresh;
      },
      () => undefined
    );
  }

  async function cancel() {
    cancelling = true;
    message = '';
    try {
      order = await apiFetch<TrackedOrder>(`${path}/cancel`, { method: 'POST' });
      confirming = false;
      pushToast('success', 'Pesanan dibatalkan.');
    } catch (error) {
      message = errorMessage(error);
    } finally {
      cancelling = false;
    }
  }

  $effect(() => {
    if (finished) return;
    return pollWhileVisible(POLL_MS, refresh);
  });

  onMount(() => {
    hydrated = true;
  });
</script>

<svelte:head>
  <title>Pesanan {order.code} | Depotin</title>
  <meta name="robots" content="noindex" />
</svelte:head>

<main class="mx-auto flex w-full max-w-md flex-col gap-5 px-4 py-6">
  <header class="flex flex-col gap-2">
    <p class="text-muted">{order.depot.name}</p>
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h1 class="text-2xl font-bold tracking-tight">Pesanan {order.code}</h1>
      <StatusBadge status={order.status} />
    </div>
    <p class="text-lg">
      Diantar <span class="font-semibold">{scheduledDayLabel(order.scheduled_date)}</span>
    </p>
    {#if !finished}
      <p class="text-muted">Diperbarui otomatis.</p>
    {/if}
  </header>

  <Card>
    <ol class="flex flex-col">
      {#each steps as step, index (step.label)}
        {@const done = step.at !== null}
        <li class="flex gap-3">
          <div class="flex flex-col items-center">
            <span
              class="mt-1 h-4 w-4 shrink-0 rounded-full {done
                ? 'bg-accent-600'
                : 'border-2 border-border'}"
            ></span>
            {#if index < steps.length - 1}
              <span class="w-0.5 flex-1 {done ? 'bg-accent-600' : 'bg-border'}"></span>
            {/if}
          </div>
          <div class="pb-5">
            <p class="font-semibold {done ? '' : 'text-muted'}">{step.label}</p>
            {#if step.at}
              <p class="text-muted">{formatDate(step.at)}, {formatTime(step.at)}</p>
            {/if}
          </div>
        </li>
      {/each}
    </ol>
  </Card>

  <Card>
    <h2 class="text-lg font-semibold">Rincian</h2>
    <ul class="mt-2 divide-y divide-border">
      {#each order.items as item (item.product_id)}
        <li class="flex items-center justify-between gap-3 py-2">
          <span>{item.qty} x {item.product_name}</span>
          <span class="tabular-nums">{formatRupiah(item.line_total)}</span>
        </li>
      {/each}
      {#if order.free_qty > 0}
        <li class="flex items-center justify-between gap-3 py-2 text-status-done">
          <span>{order.free_qty} galon gratis</span>
          <span>Rp0</span>
        </li>
      {/if}
    </ul>
    <p class="mt-3 flex items-center justify-between text-lg">
      <span>Total</span>
      <span class="text-2xl font-bold tabular-nums">{formatRupiah(order.total)}</span>
    </p>
    <p class="text-muted">{paymentLabel(order.payment_status)}</p>
    <p class="mt-3 border-t border-border pt-3">
      {order.delivery_name}
      <span class="block text-muted">{order.delivery_address}</span>
    </p>
  </Card>

  <a
    href={waLink}
    target="_blank"
    rel="external noopener"
    class="inline-flex min-h-12 items-center justify-center rounded-lg bg-accent-600 px-6 font-semibold text-white"
  >
    WhatsApp depot
  </a>

  {#if order.can_cancel}
    {#if confirming}
      <Card class="flex flex-col gap-3">
        <p class="font-semibold">Yakin batalkan pesanan ini?</p>
        {#if message}
          <Alert kind="error">{message}</Alert>
        {/if}
        <Button variant="danger" size="lg" loading={cancelling} onclick={cancel}>
          Ya, batalkan
        </Button>
        <Button variant="ghost" size="lg" onclick={() => (confirming = false)}>Tidak</Button>
      </Card>
    {:else}
      <Button variant="secondary" size="lg" disabled={!hydrated} onclick={() => (confirming = true)}>
        Batalkan pesanan
      </Button>
    {/if}
  {/if}
</main>
