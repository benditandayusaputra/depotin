<script lang="ts">
  import { onMount } from 'svelte';
  import { SvelteURLSearchParams } from 'svelte/reactivity';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import { apiFetchPage, withCursor } from '$lib/api/page';
  import type { Order } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Chip from '$lib/components/Chip.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import LinkButton from '$lib/components/LinkButton.svelte';
  import OrderCard from '$lib/components/OrderCard.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SelectField from '$lib/components/SelectField.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import type { User } from '$lib/state/session.svelte';
  import { stream } from '$lib/stream/client.svelte';
  import { isoDate } from '$lib/utils/date';
  import { errorMessage } from '$lib/utils/errors';
  import { ORDER_STATUSES, type OrderStatus } from '$lib/utils/labels';

  let date = $state(isoDate());
  let courierId = $state('');
  let status = $state<OrderStatus | ''>('');

  let orders = $state<Order[]>([]);
  let nextCursor = $state<string | null>(null);
  let couriers = $state<User[]>([]);
  let loading = $state(true);
  let loadingMore = $state(false);
  let loadError = $state('');

  const query = $derived.by(() => {
    const params = new SvelteURLSearchParams();
    if (date) params.set('date', date);
    if (courierId) params.set('courier_id', courierId);
    if (status) params.set('status', status);
    return `/orders?${params.toString()}`;
  });

  const visibleStatuses = $derived(status ? [status] : ORDER_STATUSES);
  const grouped = $derived(
    Object.fromEntries(
      visibleStatuses.map((item) => [item, orders.filter((order) => order.status === item)])
    ) as Record<OrderStatus, Order[]>
  );

  async function load(silent = false) {
    if (!silent) loading = true;
    loadError = '';
    try {
      const page = await apiFetchPage<Order>(query);
      orders = page.data;
      nextCursor = page.nextCursor;
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  async function loadMore() {
    loadingMore = true;
    try {
      const page = await apiFetchPage<Order>(withCursor(query, nextCursor));
      orders = [...orders, ...page.data];
      nextCursor = page.nextCursor;
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loadingMore = false;
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

  $effect(() => {
    void query;
    void load();
  });

  onMount(() => {
    void loadCouriers();
    return stream.subscribe(() => void load(true));
  });
</script>

<svelte:head>
  <title>Pesanan | Depotin</title>
</svelte:head>

<PageHeader title="Pesanan">
  {#snippet actions()}
    <LinkButton href={resolve('/app/pesanan/baru')}>Pesanan baru</LinkButton>
  {/snippet}
</PageHeader>

<div class="mb-4 grid gap-3 sm:grid-cols-2">
  <div class="flex flex-col gap-1">
    <label for="order-date" class="font-medium">Tanggal</label>
    <input
      id="order-date"
      type="date"
      bind:value={date}
      class="min-h-12 w-full rounded-lg border border-border bg-surface px-3 text-base text-text"
    />
  </div>
  <SelectField label="Kurir" bind:value={courierId}>
    <option value="">Semua kurir</option>
    {#each couriers as courier (courier.id)}
      <option value={courier.id}>{courier.name}</option>
    {/each}
  </SelectField>
</div>

<div
  class="-mx-4 mb-4 flex gap-2 overflow-x-auto px-4 pb-1 md:mx-0 md:flex-wrap md:px-0"
  role="group"
  aria-label="Filter status"
>
  <Chip selected={status === ''} onclick={() => (status = '')}>Semua</Chip>
  {#each ORDER_STATUSES as item (item)}
    <Chip selected={status === item} onclick={() => (status = item)}>
      <StatusBadge status={item} />
    </Chip>
  {/each}
</div>

{#if loading}
  <Skeleton lines={8} />
{:else if loadError}
  <Alert kind="error" onretry={() => void load()}>{loadError}</Alert>
{:else if orders.length === 0}
  <EmptyState
    title="Tidak ada pesanan untuk filter ini."
    description="Ubah tanggal atau buat pesanan baru."
  >
    {#snippet action()}
      <LinkButton href={resolve('/app/pesanan/baru')}>Pesanan baru</LinkButton>
    {/snippet}
  </EmptyState>
{:else}
  <div
    class="flex flex-col gap-6 md:grid md:auto-cols-[minmax(240px,1fr)] md:grid-flow-col md:items-start md:gap-3 md:overflow-x-auto md:pb-2"
  >
    {#each visibleStatuses as column (column)}
      <section aria-label="Kolom {column}" class="min-w-0">
        <h2 class="mb-2 flex items-center gap-2 text-lg font-semibold">
          <StatusBadge status={column} />
          <span>({grouped[column].length})</span>
        </h2>
        {#if grouped[column].length === 0}
          <p class="rounded-lg border border-dashed border-border px-3 py-4 text-center text-muted">
            Kosong
          </p>
        {:else}
          <ul class="flex flex-col gap-3">
            {#each grouped[column] as order (order.id)}
              <OrderCard {order} showStatus={false} />
            {/each}
          </ul>
        {/if}
      </section>
    {/each}
  </div>
  {#if nextCursor}
    <div class="mt-4 flex justify-center">
      <Button variant="secondary" loading={loadingMore} onclick={() => void loadMore()}>
        Muat lebih banyak
      </Button>
    </div>
  {/if}
{/if}
