<script lang="ts">
  import { SvelteURLSearchParams } from 'svelte/reactivity';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import { apiFetchPage, withCursor } from '$lib/api/page';
  import type { Customer, CustomerCreate } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Chip from '$lib/components/Chip.svelte';
  import CustomerForm from '$lib/components/CustomerForm.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { relativeDays } from '$lib/utils/date';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';

  const SEARCH_DEBOUNCE_MS = 300;
  const FILTERS = [
    { id: 'all', label: 'Semua' },
    { id: 'due', label: 'Jatuh tempo' },
    { id: 'at_risk', label: 'Berisiko' },
    { id: 'loan', label: 'Memegang galon' }
  ] as const;

  type FilterId = (typeof FILTERS)[number]['id'];

  let query = $state('');
  let debouncedQuery = $state('');
  let filter = $state<FilterId>('all');

  let customers = $state<Customer[]>([]);
  let nextCursor = $state<string | null>(null);
  let loading = $state(true);
  let loadingMore = $state(false);
  let loadError = $state('');

  let adding = $state(false);
  let saving = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');

  const path = $derived.by(() => {
    const params = new SvelteURLSearchParams({ filter });
    if (debouncedQuery) params.set('q', debouncedQuery);
    return `/customers?${params.toString()}`;
  });

  $effect(() => {
    const term = query.trim();
    const timer = setTimeout(() => (debouncedQuery = term), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  });

  async function load() {
    loading = true;
    loadError = '';
    try {
      const page = await apiFetchPage<Customer>(path);
      customers = page.data;
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
      const page = await apiFetchPage<Customer>(withCursor(path, nextCursor));
      customers = [...customers, ...page.data];
      nextCursor = page.nextCursor;
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loadingMore = false;
    }
  }

  async function add(values: CustomerCreate) {
    saving = true;
    errors = {};
    message = '';
    try {
      const created = await apiFetch<Customer>('/customers', { body: values });
      customers = [created, ...customers];
      adding = false;
      pushToast('success', `${created.name} ditambahkan.`);
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      saving = false;
    }
  }

  $effect(() => {
    void path;
    void load();
  });
</script>

<svelte:head>
  <title>Pelanggan | Depotin</title>
</svelte:head>

<PageHeader title="Pelanggan">
  {#snippet actions()}
    <Button onclick={() => (adding = !adding)} aria-expanded={adding}>Tambah pelanggan</Button>
  {/snippet}
</PageHeader>

{#if adding}
  <Card class="mb-4">
    <h2 class="mb-3 text-lg font-semibold">Pelanggan baru</h2>
    <CustomerForm
      submitLabel="Simpan pelanggan"
      {saving}
      {errors}
      {message}
      onsubmit={(values) => void add(values)}
      oncancel={() => (adding = false)}
    />
  </Card>
{/if}

<div class="mb-3">
  <TextField
    label="Cari pelanggan"
    bind:value={query}
    type="search"
    placeholder="Nama, nomor, atau alamat"
    autocomplete="off"
  />
</div>

<div
  class="-mx-4 mb-4 flex gap-2 overflow-x-auto px-4 pb-1 md:mx-0 md:flex-wrap md:px-0"
  role="group"
  aria-label="Filter pelanggan"
>
  {#each FILTERS as item (item.id)}
    <Chip selected={filter === item.id} onclick={() => (filter = item.id)}>{item.label}</Chip>
  {/each}
</div>

{#if loading}
  <Skeleton lines={8} />
{:else if loadError}
  <Alert kind="error" onretry={() => void load()}>{loadError}</Alert>
{:else if customers.length === 0}
  <EmptyState
    title="Belum ada pelanggan di daftar ini."
    description="Coba kata kunci lain atau tambah pelanggan baru."
  >
    {#snippet action()}
      <Button onclick={() => (adding = true)}>Tambah pelanggan</Button>
    {/snippet}
  </EmptyState>
{:else}
  <ul class="divide-y divide-border overflow-hidden rounded-lg border border-border bg-surface">
    {#each customers as customer (customer.id)}
      <li>
        <a
          href={resolve('/app/pelanggan/[id]', { id: customer.id })}
          class="flex min-h-16 flex-col gap-1 px-4 py-3 hover:bg-neutral-100 dark:hover:bg-neutral-800"
        >
          <div class="flex items-start justify-between gap-3">
            <p class="truncate font-semibold">{customer.name}</p>
            {#if customer.loan_balance > 0}
              <Badge tone="pending">{customer.loan_balance} galon dipinjam</Badge>
            {/if}
          </div>
          <p class="text-sm text-muted">
            {customer.phone}{customer.area ? ` · ${customer.area}` : ''}
          </p>
          {#if customer.predicted_empty_at}
            <p class="text-sm text-muted">
              Diperkirakan habis {relativeDays(customer.predicted_empty_at)}
            </p>
          {/if}
        </a>
      </li>
    {/each}
  </ul>
  {#if nextCursor}
    <div class="mt-4 flex justify-center">
      <Button variant="secondary" loading={loadingMore} onclick={() => void loadMore()}>
        Muat lebih banyak
      </Button>
    </div>
  {/if}
{/if}
