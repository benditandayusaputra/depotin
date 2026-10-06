<script lang="ts">
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import type { ReportSummary } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Card from '$lib/components/Card.svelte';
  import Chip from '$lib/components/Chip.svelte';
  import LinkButton from '$lib/components/LinkButton.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import RevenueChart from '$lib/components/RevenueChart.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import { isoDate, shiftDays } from '$lib/utils/date';
  import { errorMessage } from '$lib/utils/errors';
  import { orderSourceLabel, type OrderSource } from '$lib/utils/labels';
  import { formatRupiah } from '$lib/utils/rupiah';

  const SOURCES: OrderSource[] = ['public', 'link', 'reminder', 'owner', 'courier'];
  const today = isoDate();
  const monthStart = `${today.slice(0, 7)}-01`;

  let from = $state(shiftDays(today, -29));
  let to = $state(today);
  let summary = $state<ReportSummary | null>(null);
  let loading = $state(true);
  let loadError = $state('');

  const rangeQuery = $derived(`from=${from}&to=${to}`);
  const exportHref = $derived(
    `${resolve('/api/[...path]', { path: 'v1/reports/export.csv' })}?${rangeQuery}`
  );
  const conversionPercent = $derived(
    summary ? Math.round(summary.conversion_rate * 100).toString() : '0'
  );

  function preset(days: number) {
    to = today;
    from = shiftDays(today, -(days - 1));
  }

  function thisMonth() {
    from = monthStart;
    to = today;
  }

  async function load() {
    if (!from || !to) return;
    loading = true;
    loadError = '';
    try {
      summary = await apiFetch<ReportSummary>(`/reports/summary?${rangeQuery}`);
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void rangeQuery;
    void load();
  });
</script>

<svelte:head>
  <title>Laporan | Depotin</title>
</svelte:head>

<PageHeader title="Laporan">
  {#snippet actions()}
    <LinkButton href={exportHref} variant="secondary">Ekspor CSV</LinkButton>
  {/snippet}
</PageHeader>

<div class="mb-3 grid grid-cols-2 gap-3">
  <div class="flex flex-col gap-1">
    <label for="report-from" class="font-medium">Dari</label>
    <input
      id="report-from"
      type="date"
      bind:value={from}
      max={to}
      class="min-h-12 w-full rounded-lg border border-border bg-surface px-3 text-base text-text"
    />
  </div>
  <div class="flex flex-col gap-1">
    <label for="report-to" class="font-medium">Sampai</label>
    <input
      id="report-to"
      type="date"
      bind:value={to}
      min={from}
      max={today}
      class="min-h-12 w-full rounded-lg border border-border bg-surface px-3 text-base text-text"
    />
  </div>
</div>

<div class="mb-4 flex flex-wrap gap-2" role="group" aria-label="Rentang cepat">
  <Chip selected={from === shiftDays(today, -6) && to === today} onclick={() => preset(7)}>
    7 hari
  </Chip>
  <Chip selected={from === shiftDays(today, -29) && to === today} onclick={() => preset(30)}>
    30 hari
  </Chip>
  <Chip selected={from === monthStart && to === today} onclick={thisMonth}>Bulan ini</Chip>
</div>

{#if loading}
  <Skeleton lines={8} />
{:else if loadError || !summary}
  <Alert kind="error" onretry={() => void load()}>{loadError}</Alert>
{:else}
  <div class="flex flex-col gap-4">
    <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
      <Card class="col-span-2 md:col-span-1">
        <p class="text-sm text-muted">Omzet</p>
        <p class="text-num">{formatRupiah(summary.revenue)}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Galon terjual</p>
        <p class="text-num">{summary.gallons_sold}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Pesanan selesai</p>
        <p class="text-num">{summary.delivered_orders}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Pelanggan baru</p>
        <p class="text-num">{summary.new_customers}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Pelanggan aktif</p>
        <p class="text-num">{summary.active_customers}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Konversi pengingat</p>
        <p class="text-num">{conversionPercent}%</p>
        <p class="text-sm text-muted">
          {summary.reminders_converted} dari {summary.reminders_sent} terkirim jadi pesanan
        </p>
      </Card>
    </div>

    <Card>
      <h2 class="mb-3 text-lg font-semibold">Omzet harian</h2>
      {#if summary.daily.length === 0}
        <p class="text-muted">Belum ada pesanan selesai di rentang ini.</p>
      {:else}
        <RevenueChart daily={summary.daily} />
      {/if}
    </Card>

    <Card>
      <h2 class="mb-3 text-lg font-semibold">Pesanan per sumber</h2>
      <ul class="divide-y divide-border">
        {#each SOURCES as source (source)}
          <li class="flex items-center justify-between gap-3 py-2">
            <span>{orderSourceLabel(source)}</span>
            <span class="text-lg font-bold">{summary.orders_by_source[source] ?? 0}</span>
          </li>
        {/each}
      </ul>
    </Card>
  </div>
{/if}
