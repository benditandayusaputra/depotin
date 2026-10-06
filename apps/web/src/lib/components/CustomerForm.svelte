<script lang="ts">
  import { untrack } from 'svelte';
  import type { Customer, CustomerCreate } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import QtyStepper from '$lib/components/QtyStepper.svelte';
  import TextField from '$lib/components/TextField.svelte';

  let {
    initial,
    submitLabel,
    saving,
    errors,
    message,
    onsubmit,
    oncancel
  }: {
    initial?: Customer;
    submitLabel: string;
    saving: boolean;
    errors: Record<string, string>;
    message: string;
    onsubmit: (values: CustomerCreate) => void;
    oncancel?: () => void;
  } = $props();

  const start = untrack(() => initial);

  let name = $state(start?.name ?? '');
  let phone = $state(start?.phone ?? '');
  let address = $state(start?.address ?? '');
  let addressNote = $state(start?.address_note ?? '');
  let area = $state(start?.area ?? '');
  let usualQty = $state(start?.usual_qty ?? 1);

  function submit(event: SubmitEvent) {
    event.preventDefault();
    onsubmit({
      name: name.trim(),
      phone: phone.trim(),
      address: address.trim(),
      address_note: addressNote.trim(),
      area: area.trim(),
      usual_qty: usualQty
    });
  }
</script>

<form class="flex flex-col gap-4" onsubmit={submit} novalidate>
  <TextField label="Nama pelanggan" bind:value={name} error={errors.name} required />
  <TextField
    label="Nomor WhatsApp"
    bind:value={phone}
    error={errors.phone}
    type="tel"
    inputmode="tel"
    placeholder="08xxxxxxxxxx"
    required
  />
  <TextField label="Alamat" bind:value={address} error={errors.address} />
  <TextField
    label="Patokan"
    bind:value={addressNote}
    error={errors.address_note}
    placeholder="Misal: sebelah warung"
  />
  <TextField label="Area" bind:value={area} error={errors.area} placeholder="Misal: RT 01" />
  <QtyStepper label="Jumlah biasa" bind:value={usualQty} min={1} error={errors.usual_qty} />
  {#if message}
    <Alert kind="error">{message}</Alert>
  {/if}
  <div class="flex flex-wrap gap-2">
    <Button type="submit" size="lg" loading={saving}>{submitLabel}</Button>
    {#if oncancel}
      <Button variant="ghost" disabled={saving} onclick={oncancel}>Batal</Button>
    {/if}
  </div>
</form>
