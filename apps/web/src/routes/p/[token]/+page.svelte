<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { resolve } from '$app/paths';
  import { ApiError, apiFetch, newIdempotencyKey } from '$lib/api/client';
  import type { components } from '$lib/api/schema';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import QtyStepper from '$lib/components/QtyStepper.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { formatDate } from '$lib/utils/date';
  import { errorMessage } from '$lib/utils/errors';
  import { pollWhileVisible } from '$lib/utils/poll';
  import { formatRupiah } from '$lib/utils/rupiah';
  import { jakartaDate, scheduledDayLabel } from '$lib/utils/schedule';
  import type { PageProps } from './$types';

  type PersonalPage = components['schemas']['PersonalPage'];
  type CreatedOrder = components['schemas']['CreatedOrder'];
  type Day = 'today' | 'tomorrow';

  const POLL_MS = 15_000;
  const DAYS: { value: Day; label: string }[] = [
    { value: 'today', label: 'Hari ini' },
    { value: 'tomorrow', label: 'Besok' }
  ];

  let { data }: PageProps = $props();

  let me = $derived(data.me);
  let qty = $state(untrack(() => data.me.customer.usual_qty));
  let day = $state<Day>('today');
  let note = $state('');
  let submitting = $state(false);
  let hydrated = $state(false);
  let message = $state('');
  let created = $state<CreatedOrder | null>(null);
  let idempotencyKey = $state(newIdempotencyKey());

  const path = $derived(`/public/me/${encodeURIComponent(data.token)}`);
  const buttonLabel = $derived(
    qty === me.customer.usual_qty ? `Pesan ${qty} galon seperti biasa` : `Pesan ${qty} galon`
  );
  const estimate = $derived(qty * me.depot.refill_price + me.depot.delivery_fee);
  const hasActive = $derived(me.active_orders.length > 0);

  function refresh() {
    void apiFetch<PersonalPage>(path).then(
      (fresh) => {
        me = fresh;
      },
      () => undefined
    );
  }

  async function order() {
    submitting = true;
    message = '';
    try {
      created = await apiFetch<CreatedOrder>(`${path}/orders`, {
        body: {
          qty,
          scheduled_date: jakartaDate(day === 'today' ? 0 : 1),
          note: note || undefined,
          r: data.reminder ?? undefined
        },
        idempotencyKey
      });
      pushToast('success', 'Pesanan diterima.');
      refresh();
    } catch (error) {
      if (error instanceof ApiError) idempotencyKey = newIdempotencyKey();
      message = errorMessage(error);
    } finally {
      submitting = false;
    }
  }

  function orderAgain() {
    created = null;
    idempotencyKey = newIdempotencyKey();
  }

  $effect(() => {
    if (!hasActive) return;
    return pollWhileVisible(POLL_MS, refresh);
  });

  onMount(() => {
    hydrated = true;
  });
</script>

<svelte:head>
  <title>Halaman {me.customer.name} | Depotin</title>
  <meta name="robots" content="noindex" />
</svelte:head>

<main class="mx-auto flex w-full max-w-md flex-col gap-5 px-4 py-6">
  <header>
    <h1 class="text-3xl font-bold tracking-tight">Halo, {me.customer.name}</h1>
    <p class="text-muted">{me.depot.name}, nomor {me.customer.phone_masked}</p>
  </header>

  {#if created}
    <Card class="flex flex-col gap-3 border-status-done bg-status-done-soft">
      <h2 class="text-2xl font-bold text-status-done">Pesanan diterima</h2>
      <p class="text-lg">
        Kode <span class="font-bold">{created.code}</span>
      </p>
      <StatusBadge status={created.status} class="w-fit" />
      <a
        href={resolve('/t/[token]', { token: created.track_token })}
        class="inline-flex min-h-12 items-center justify-center rounded-lg bg-accent-600 px-6 font-semibold text-white"
      >
        Lacak pesanan
      </a>
      <Button variant="ghost" size="lg" onclick={orderAgain}>Pesan lagi</Button>
    </Card>
  {:else}
    <section class="flex flex-col gap-4">
      <Button
        size="lg"
        class="min-h-16 w-full text-xl"
        loading={submitting}
        disabled={!hydrated}
        onclick={order}
      >
        {buttonLabel}
      </Button>
      <p class="text-center text-muted">
        Perkiraan total <span class="font-semibold text-text">{formatRupiah(estimate)}</span>
      </p>
      {#if message}
        <Alert kind="error" onretry={order}>{message}</Alert>
      {/if}
      <QtyStepper label="Jumlah galon" bind:value={qty} />
      <div class="flex items-center justify-between gap-3">
        <span class="font-medium">Hari antar</span>
        <div class="flex gap-2" role="group" aria-label="Hari antar">
          {#each DAYS as option (option.value)}
            <button
              type="button"
              aria-pressed={day === option.value}
              class="min-h-12 rounded-full border-2 px-4 font-semibold {day === option.value
                ? 'border-accent-600 bg-accent-50 text-accent-700 dark:bg-accent-900/40 dark:text-accent-300'
                : 'border-border bg-surface'}"
              onclick={() => (day = option.value)}
            >
              {option.label}
            </button>
          {/each}
        </div>
      </div>
      <TextField label="Catatan (opsional)" bind:value={note} placeholder="Taruh di depan pagar" />
    </section>
  {/if}

  {#if hasActive}
    <section class="flex flex-col gap-3">
      <h2 class="text-lg font-semibold">Pesanan aktif</h2>
      {#each me.active_orders as active (active.code)}
        <Card class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <p class="font-bold">{active.code}</p>
            <StatusBadge status={active.status} />
          </div>
          <p>
            {active.refill_qty} galon, diantar {scheduledDayLabel(active.scheduled_date)}
          </p>
          <p class="font-semibold tabular-nums">{formatRupiah(active.total)}</p>
        </Card>
      {/each}
    </section>
  {/if}

  <div class="grid gap-3 sm:grid-cols-2">
    <Card>
      <p class="text-muted">Galon depot di rumah</p>
      <p class="text-num tabular-nums">{me.customer.loan_balance}</p>
    </Card>
    {#if me.depot.loyalty_every !== null}
      <Card>
        <p class="text-muted">Stempel {me.customer.stamp_count} dari {me.depot.loyalty_every}</p>
        <div class="mt-2 flex flex-wrap gap-2" aria-hidden="true">
          {#each { length: me.depot.loyalty_every }, index (index)}
            <span
              class="h-5 w-5 rounded-full {index < me.customer.stamp_count
                ? 'bg-accent-600'
                : 'border-2 border-border'}"
            ></span>
          {/each}
        </div>
        <p class="mt-2 text-muted">Isi {me.depot.loyalty_every} gratis 1.</p>
      </Card>
    {/if}
  </div>

  <section class="flex flex-col gap-3">
    <h2 class="text-lg font-semibold">Pesanan terakhir</h2>
    {#if me.recent_orders.length === 0}
      <p class="text-muted">Belum ada riwayat.</p>
    {:else}
      <ul class="divide-y divide-border rounded-lg border border-border bg-surface">
        {#each me.recent_orders as recent (recent.code)}
          <li class="flex items-center justify-between gap-3 px-4 py-3">
            <span>
              {recent.refill_qty} galon
              <span class="block text-muted">{formatDate(recent.created_at)}</span>
            </span>
            <span class="flex flex-col items-end gap-1">
              <span class="font-semibold tabular-nums">{formatRupiah(recent.total)}</span>
              <StatusBadge status={recent.status} />
            </span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</main>
