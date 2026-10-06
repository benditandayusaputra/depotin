<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import type { GallonSummary } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import { relativeDays } from '$lib/utils/date';
  import { errorMessage } from '$lib/utils/errors';

  let summary = $state<GallonSummary | null>(null);
  let loading = $state(true);
  let loadError = $state('');

  async function load() {
    loading = true;
    loadError = '';
    try {
      summary = await apiFetch<GallonSummary>('/gallons/summary');
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<svelte:head>
  <title>Galon | Depotin</title>
</svelte:head>

<PageHeader title="Galon" description="Galon depot yang sedang berada di pelanggan." />

{#if loading}
  <Skeleton lines={6} />
{:else if loadError || !summary}
  <Alert kind="error" onretry={() => void load()}>{loadError}</Alert>
{:else}
  <div class="mb-5 grid grid-cols-2 gap-3">
    <Card>
      <p class="text-sm text-muted">Total galon dipinjam</p>
      <p class="text-num">{summary.total_on_loan}</p>
    </Card>
    <Card>
      <p class="text-sm text-muted">Pelanggan memegang galon</p>
      <p class="text-num">{summary.customers_with_loan}</p>
    </Card>
  </div>

  <h2 class="mb-3 text-lg font-semibold">Galon mengendap ({summary.idle.length})</h2>
  <p class="mb-3 text-muted">Memegang galon tetapi tidak memesan lebih dari 30 hari.</p>
  {#if summary.idle.length === 0}
    <EmptyState title="Tidak ada galon yang mengendap." />
  {:else}
    <ul class="divide-y divide-border overflow-hidden rounded-lg border border-border bg-surface">
      {#each summary.idle as customer (customer.id)}
        <li>
          <a
            href={resolve('/app/pelanggan/[id]', { id: customer.id })}
            class="flex min-h-16 items-center justify-between gap-3 px-4 py-3 hover:bg-neutral-100 dark:hover:bg-neutral-800"
          >
            <div class="min-w-0">
              <p class="truncate font-semibold">{customer.name}</p>
              <p class="text-sm text-muted">
                {customer.last_delivered_at
                  ? `Terakhir diantar ${relativeDays(customer.last_delivered_at)}`
                  : 'Belum pernah diantar'}
              </p>
            </div>
            <Badge tone="pending">{customer.loan_balance} galon</Badge>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
{/if}
