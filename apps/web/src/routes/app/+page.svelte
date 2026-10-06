<script lang="ts">
  import { onMount } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import { apiFetchPage } from '$lib/api/page';
  import type { DashboardToday, Order } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import LinkButton from '$lib/components/LinkButton.svelte';
  import OrderCard from '$lib/components/OrderCard.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { stream } from '$lib/stream/client.svelte';
  import { formatDate, isoDate } from '$lib/utils/date';
  import { errorMessage } from '$lib/utils/errors';
  import { ORDER_STATUSES } from '$lib/utils/labels';
  import { formatRupiah } from '$lib/utils/rupiah';

  let summary = $state<DashboardToday | null>(null);
  let pending = $state<Order[]>([]);
  let active = $state<Order[]>([]);
  let loading = $state(true);
  let loadError = $state('');
  const confirming = new SvelteSet<string>();
  let confirmError = $state('');

  async function load(silent = false) {
    if (!silent) loading = true;
    loadError = '';
    try {
      const [today, pendingPage, todayPage] = await Promise.all([
        apiFetch<DashboardToday>('/dashboard/today'),
        apiFetchPage<Order>('/orders?status=pending'),
        apiFetchPage<Order>(`/orders?date=${isoDate()}`)
      ]);
      summary = today;
      pending = pendingPage.data;
      active = todayPage.data.filter(
        (order) => order.status === 'confirmed' || order.status === 'on_delivery'
      );
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  async function confirm(order: Order) {
    confirming.add(order.id);
    confirmError = '';
    pending = pending.filter((item) => item.id !== order.id);
    active = [{ ...order, status: 'confirmed' }, ...active];
    try {
      const updated = await apiFetch<Order>(`/orders/${order.id}/confirm`, { method: 'POST' });
      active = active.map((item) => (item.id === updated.id ? updated : item));
      pushToast('success', `Pesanan ${order.code} dikonfirmasi.`);
    } catch (error) {
      active = active.filter((item) => item.id !== order.id);
      pending = [order, ...pending];
      confirmError = errorMessage(error);
    } finally {
      confirming.delete(order.id);
    }
  }

  onMount(() => {
    void load();
    return stream.subscribe(() => void load(true));
  });
</script>

<svelte:head>
  <title>Hari ini | Depotin</title>
</svelte:head>

<PageHeader title="Hari ini" description={formatDate(new Date().toISOString())}>
  {#snippet actions()}
    <LinkButton href={resolve('/app/pesanan/baru')}>Pesanan baru</LinkButton>
  {/snippet}
</PageHeader>

{#if loading}
  <Skeleton lines={8} />
{:else if loadError}
  <Alert kind="error" onretry={() => void load()}>{loadError}</Alert>
{:else if summary}
  <div class="flex flex-col gap-5">
    <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
      <Card class="col-span-2 md:col-span-1">
        <p class="text-sm text-muted">Omzet hari ini</p>
        <p class="text-num">{formatRupiah(summary.revenue)}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Galon terjual</p>
        <p class="text-num">{summary.gallons_sold}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Perkiraan pesanan hari ini</p>
        <p class="text-num">{summary.expected_demand}</p>
        <p class="text-sm text-muted">galon dari {summary.customers_due} pelanggan</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Pengingat menunggu</p>
        <p class="text-num">{summary.reminders_queued}</p>
        <a
          href={resolve('/app/pengingat')}
          class="text-sm font-semibold text-accent-700 underline dark:text-accent-300"
        >
          Buka pengingat
        </a>
      </Card>
      <Card>
        <p class="text-sm text-muted">Galon dipinjam</p>
        <p class="text-num">{summary.gallons_on_loan}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Pesanan selesai</p>
        <p class="text-num">{summary.delivered_orders}</p>
      </Card>
    </div>

    <Card>
      <h2 class="mb-3 text-lg font-semibold">Pesanan hari ini per status</h2>
      <ul class="flex flex-wrap gap-2">
        {#each ORDER_STATUSES as status (status)}
          <li class="flex items-center gap-2 rounded-lg border border-border px-3 py-2">
            <StatusBadge {status} />
            <span class="text-lg font-bold">{summary.orders_by_status[status] ?? 0}</span>
          </li>
        {/each}
      </ul>
    </Card>

    <section>
      <h2 class="mb-3 text-lg font-semibold">Menunggu konfirmasi ({pending.length})</h2>
      {#if confirmError}
        <div class="mb-3"><Alert kind="error">{confirmError}</Alert></div>
      {/if}
      {#if pending.length === 0}
        <EmptyState title="Tidak ada pesanan yang menunggu." />
      {:else}
        <ul class="flex flex-col gap-3">
          {#each pending as order (order.id)}
            <OrderCard {order} showStatus={false}>
              {#snippet actions()}
                <Button
                  size="lg"
                  class="flex-1"
                  loading={confirming.has(order.id)}
                  onclick={() => void confirm(order)}
                >
                  Konfirmasi
                </Button>
              {/snippet}
            </OrderCard>
          {/each}
        </ul>
      {/if}
    </section>

    <section>
      <h2 class="mb-3 text-lg font-semibold">Antrean antar hari ini ({active.length})</h2>
      {#if active.length === 0}
        <EmptyState title="Belum ada pesanan yang siap diantar." />
      {:else}
        <ul class="flex flex-col gap-3">
          {#each active as order (order.id)}
            <OrderCard {order} />
          {/each}
        </ul>
      {/if}
    </section>

    <div class="flex flex-col gap-2 sm:flex-row">
      <LinkButton href={resolve('/app/pesanan/baru')} class="flex-1">Pesanan baru</LinkButton>
      <LinkButton href={resolve('/app/pengingat')} variant="secondary" class="flex-1">
        Buka pengingat
      </LinkButton>
    </div>
  </div>
{/if}
