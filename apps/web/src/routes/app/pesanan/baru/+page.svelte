<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { apiFetch, newIdempotencyKey } from '$lib/api/client';
  import { apiFetchPage } from '$lib/api/page';
  import type { Customer, Order, OrderCreate, Product } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Chip from '$lib/components/Chip.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import QtyStepper from '$lib/components/QtyStepper.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { session } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { isoDate, shiftDays } from '$lib/utils/date';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';
  import { fulfilmentLabel, type Fulfilment } from '$lib/utils/labels';
  import { formatRupiah } from '$lib/utils/rupiah';

  const SEARCH_DEBOUNCE_MS = 300;
  const idempotencyKey = newIdempotencyKey();
  const today = isoDate();
  const tomorrow = shiftDays(today, 1);

  let query = $state('');
  let results = $state<Customer[]>([]);
  let searching = $state(false);
  let searchError = $state('');
  let customer = $state<Customer | null>(null);

  let qty = $state(1);
  let fulfilment = $state<Fulfilment>('delivery');
  let scheduledDate = $state(today);
  let note = $state('');

  let refillPrice = $state<number | null>(null);
  let saving = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');

  const previewTotal = $derived(
    refillPrice === null
      ? null
      : qty * refillPrice + (fulfilment === 'delivery' ? (session.depot?.delivery_fee ?? 0) : 0)
  );

  $effect(() => {
    const term = query.trim();
    if (customer || term.length === 0) {
      results = [];
      return;
    }
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      searching = true;
      searchError = '';
      try {
        const page = await apiFetchPage<Customer>(`/customers?q=${encodeURIComponent(term)}`);
        if (!controller.signal.aborted) results = page.data;
      } catch (error) {
        if (!controller.signal.aborted) searchError = errorMessage(error);
      } finally {
        if (!controller.signal.aborted) searching = false;
      }
    }, SEARCH_DEBOUNCE_MS);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  });

  function pick(selected: Customer) {
    customer = selected;
    qty = selected.usual_qty;
    results = [];
    errors = {};
  }

  function clearCustomer() {
    customer = null;
    query = '';
  }

  async function loadRefillPrice() {
    try {
      const products = await apiFetch<Product[]>('/products');
      refillPrice =
        products.find((item) => item.kind === 'refill' && item.is_active)?.price ?? null;
    } catch {
      refillPrice = null;
    }
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (!customer) {
      errors = { customer_id: 'Pilih pelanggan dulu.' };
      return;
    }
    saving = true;
    errors = {};
    message = '';
    const body: OrderCreate = {
      customer_id: customer.id,
      refill_qty: qty,
      fulfilment,
      scheduled_date: scheduledDate
    };
    if (note.trim()) body.note = note.trim();
    try {
      const order = await apiFetch<Order>('/orders', { body, idempotencyKey });
      pushToast('success', `Pesanan ${order.code} tersimpan. Total ${formatRupiah(order.total)}.`);
      await goto(resolve('/app/pesanan/[id]', { id: order.id }));
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      saving = false;
    }
  }

  onMount(() => {
    void loadRefillPrice();
  });
</script>

<svelte:head>
  <title>Pesanan baru | Depotin</title>
</svelte:head>

<PageHeader title="Pesanan baru" description="Satu layar, langsung simpan." />

<form class="flex flex-col gap-4" onsubmit={save} novalidate>
  <Card>
    <h2 class="mb-3 text-lg font-semibold">Pelanggan</h2>
    {#if customer}
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <p class="font-semibold">{customer.name}</p>
          <p class="text-muted">{customer.phone}</p>
          <p class="text-muted">{customer.address}</p>
        </div>
        <Button variant="ghost" onclick={clearCustomer}>Ganti</Button>
      </div>
    {:else}
      <TextField
        label="Cari pelanggan"
        bind:value={query}
        placeholder="Ketik nama, nomor, atau alamat"
        autocomplete="off"
        error={errors.customer_id}
        hint={searching ? 'Mencari...' : undefined}
      />
      {#if searchError}
        <div class="mt-2"><Alert kind="error">{searchError}</Alert></div>
      {:else if results.length > 0}
        <ul
          class="mt-2 divide-y divide-border rounded-lg border border-border"
          aria-label="Hasil pencarian"
        >
          {#each results as item (item.id)}
            <li>
              <button
                type="button"
                class="flex min-h-14 w-full flex-col items-start px-3 py-2 text-left hover:bg-neutral-100 dark:hover:bg-neutral-800"
                onclick={() => pick(item)}
              >
                <span class="font-semibold">{item.name}</span>
                <span class="text-sm text-muted">{item.phone} &middot; {item.address}</span>
              </button>
            </li>
          {/each}
        </ul>
      {:else if query.trim() && !searching}
        <p class="mt-2 text-muted">Tidak ada pelanggan yang cocok.</p>
      {/if}
    {/if}
  </Card>

  <Card>
    <h2 class="mb-3 text-lg font-semibold">Rincian</h2>
    <div class="flex flex-col gap-4">
      <QtyStepper label="Jumlah galon" bind:value={qty} min={1} error={errors.refill_qty} />

      <fieldset>
        <legend class="mb-2 font-medium">Cara ambil</legend>
        <div class="flex gap-2">
          {#each ['delivery', 'pickup'] as const as option (option)}
            <Chip selected={fulfilment === option} onclick={() => (fulfilment = option)}>
              {fulfilmentLabel(option)}
            </Chip>
          {/each}
        </div>
      </fieldset>

      <fieldset>
        <legend class="mb-2 font-medium">Hari antar</legend>
        <div class="flex gap-2">
          <Chip selected={scheduledDate === today} onclick={() => (scheduledDate = today)}>
            Hari ini
          </Chip>
          <Chip selected={scheduledDate === tomorrow} onclick={() => (scheduledDate = tomorrow)}>
            Besok
          </Chip>
        </div>
        {#if errors.scheduled_date}
          <p class="mt-1 text-status-cancelled">{errors.scheduled_date}</p>
        {/if}
      </fieldset>

      <TextField label="Catatan" bind:value={note} placeholder="Opsional" error={errors.note} />
    </div>
  </Card>

  <Card>
    <div class="flex items-center justify-between gap-3">
      <div>
        <p class="font-medium">Perkiraan total</p>
        <p class="text-sm text-muted">Total pasti dihitung server saat disimpan.</p>
      </div>
      <p class="text-2xl font-bold">{previewTotal === null ? '-' : formatRupiah(previewTotal)}</p>
    </div>
  </Card>

  {#if message}
    <Alert kind="error">{message}</Alert>
  {/if}

  <Button type="submit" size="lg" loading={saving} disabled={!customer}>Simpan pesanan</Button>
</form>
