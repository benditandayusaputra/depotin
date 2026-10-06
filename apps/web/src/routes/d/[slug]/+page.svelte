<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { ApiError, apiFetch, newIdempotencyKey } from '$lib/api/client';
  import type { components } from '$lib/api/schema';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import CounterStepper from '$lib/components/CounterStepper.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';
  import { productKindLabel } from '$lib/utils/labels';
  import { normalizePhone } from '$lib/utils/phone';
  import { formatRupiah } from '$lib/utils/rupiah';
  import { waUrl } from '$lib/utils/whatsapp';
  import type { PageProps } from './$types';

  type CreatedOrder = components['schemas']['CreatedOrder'];

  let { data }: PageProps = $props();

  let name = $state('');
  let phone = $state('');
  let address = $state('');
  let addressNote = $state('');
  let note = $state('');
  let qty = $state(1);
  let website = $state('');
  let submitting = $state(false);
  let hydrated = $state(false);
  let message = $state('');
  let errors = $state<Record<string, string>>({});
  let idempotencyKey = $state(newIdempotencyKey());
  let form = $state<HTMLFormElement>();

  const refill = $derived(data.products.find((product) => product.kind === 'refill'));
  const estimate = $derived((refill?.price ?? 0) * qty + data.depot.delivery_fee);
  const waLink = $derived(
    waUrl(data.depot.phone, `Halo ${data.depot.name}, saya mau pesan galon.`)
  );

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    message = '';
    errors = {};
    const normalizedPhone = normalizePhone(phone);
    if (normalizedPhone === null) {
      errors = { phone: 'Nomor WhatsApp tidak valid.' };
      return;
    }
    submitting = true;
    try {
      const created = await apiFetch<CreatedOrder>(`/public/depots/${data.depot.slug}/orders`, {
        body: {
          name,
          phone: normalizedPhone,
          address,
          address_note: addressNote || undefined,
          qty,
          note: note || undefined,
          website
        },
        idempotencyKey
      });
      await goto(resolve('/t/[token]', { token: created.track_token }));
    } catch (error) {
      if (error instanceof ApiError) idempotencyKey = newIdempotencyKey();
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      submitting = false;
    }
  }

  onMount(() => {
    hydrated = true;
  });
</script>

<svelte:head>
  <title>{data.depot.name} | Depotin</title>
  <meta name="description" content="Pesan galon dari {data.depot.name} lewat WhatsApp atau form." />
</svelte:head>

<main class="mx-auto flex w-full max-w-md flex-col gap-5 px-4 py-6">
  <header class="flex flex-col gap-2">
    <h1 class="text-3xl font-bold tracking-tight">{data.depot.name}</h1>
    <p class="text-muted">{data.depot.address}</p>
    <p>Buka {data.depot.open_time} sampai {data.depot.close_time}</p>
    {#if data.depot.is_accepting_orders}
      <p
        class="inline-flex w-fit items-center rounded-md bg-status-done-soft px-2.5 py-1 font-semibold text-status-done"
      >
        Menerima pesanan
      </p>
    {:else}
      <p
        class="inline-flex w-fit items-center rounded-md bg-status-cancelled-soft px-2.5 py-1 font-semibold text-status-cancelled"
      >
        Sedang tutup
      </p>
    {/if}
  </header>

  <Card>
    <h2 class="text-lg font-semibold">Harga</h2>
    <ul class="mt-2 divide-y divide-border">
      {#each data.products as product (product.id)}
        <li class="flex items-center justify-between gap-3 py-2">
          <span>
            {product.name}
            <span class="block text-muted">{productKindLabel(product.kind)}</span>
          </span>
          <span class="font-semibold tabular-nums">{formatRupiah(product.price)}</span>
        </li>
      {/each}
    </ul>
    <p class="mt-3 text-muted">
      Ongkos kirim {data.depot.delivery_fee === 0
        ? 'gratis'
        : formatRupiah(data.depot.delivery_fee)}
    </p>
  </Card>

  {#if data.depot.is_accepting_orders}
    <form bind:this={form} class="flex flex-col gap-4" onsubmit={submit} novalidate>
      <h2 class="text-xl font-semibold">Pesan galon</h2>
      <TextField label="Nama" bind:value={name} autocomplete="name" error={errors.name} required />
      <TextField
        label="Nomor WhatsApp"
        bind:value={phone}
        type="tel"
        inputmode="tel"
        autocomplete="tel"
        placeholder="08xxxxxxxxxx"
        error={errors.phone}
        required
      />
      <div class="flex flex-col gap-1">
        <label for="address" class="font-medium">Alamat</label>
        <textarea
          id="address"
          bind:value={address}
          rows="3"
          autocomplete="street-address"
          aria-invalid={errors.address ? true : undefined}
          class="w-full rounded-lg border bg-surface px-3 py-2 text-base text-text {errors.address
            ? 'border-status-cancelled'
            : 'border-border'}"
          required></textarea>
        {#if errors.address}
          <p class="text-status-cancelled">{errors.address}</p>
        {/if}
      </div>
      <TextField
        label="Patokan (opsional)"
        bind:value={addressNote}
        placeholder="Dekat masjid, pagar hijau"
        error={errors.address_note}
      />
      <CounterStepper label="Jumlah galon" bind:value={qty} />
      {#if errors.qty}
        <p class="text-status-cancelled">{errors.qty}</p>
      {/if}
      <TextField label="Catatan (opsional)" bind:value={note} error={errors.note} />
      <input
        name="website"
        bind:value={website}
        class="sr-only"
        tabindex="-1"
        autocomplete="off"
        aria-hidden="true"
      />

      <p class="flex items-center justify-between text-lg">
        <span>Perkiraan total</span>
        <span class="font-bold tabular-nums">{formatRupiah(estimate)}</span>
      </p>

      {#if message}
        <Alert kind="error" onretry={() => form?.requestSubmit()}>{message}</Alert>
      {/if}

      <Button type="submit" size="lg" loading={submitting} disabled={!hydrated}>
        Pesan sekarang
      </Button>
    </form>
  {:else}
    <Alert kind="info">Depot sedang tidak menerima pesanan. Hubungi lewat WhatsApp.</Alert>
  {/if}

  <a
    href={waLink}
    target="_blank"
    rel="external noopener"
    class="inline-flex min-h-12 items-center justify-center rounded-lg border border-border bg-surface px-6 font-semibold text-accent-700 dark:text-accent-300"
  >
    WhatsApp depot
  </a>
</main>
