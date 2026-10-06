<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import type { components } from '$lib/api/schema';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { session, type Depot } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';

  type Product = components['schemas']['Product'];

  let loading = $state(true);
  let loadError = $state('');
  let refill = $state<Product | null>(null);
  let refillPrice = $state('');
  let openTime = $state('');
  let closeTime = $state('');
  let deliveryFee = $state('');
  let address = $state('');
  let saving = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');

  async function load() {
    loading = true;
    loadError = '';
    try {
      const [products, depot] = await Promise.all([
        apiFetch<Product[]>('/products'),
        apiFetch<Depot>('/depot')
      ]);
      refill = products.find((product) => product.kind === 'refill') ?? null;
      refillPrice = refill ? String(refill.price) : '';
      openTime = depot.open_time;
      closeTime = depot.close_time;
      deliveryFee = String(depot.delivery_fee);
      address = depot.address;
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    saving = true;
    errors = {};
    message = '';
    try {
      if (refill && Number(refillPrice) !== refill.price) {
        refill = await apiFetch<Product>(`/products/${refill.id}`, {
          method: 'PATCH',
          body: { price: Number(refillPrice) }
        });
      }
      await session.saveDepot({
        open_time: openTime,
        close_time: closeTime,
        delivery_fee: Number(deliveryFee),
        address
      });
      pushToast('success', 'Pengaturan awal tersimpan.');
      await goto(resolve('/app'));
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      saving = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<svelte:head>
  <title>Penyiapan awal | Depotin</title>
</svelte:head>

<PageHeader
  title="Siapkan depot Anda"
  description="Isi beberapa hal dasar supaya pesanan bisa langsung diterima. Semua bisa diubah nanti."
/>

<Card>
  {#if loading}
    <Skeleton lines={6} />
  {:else if loadError}
    <Alert kind="error" onretry={load}>{loadError}</Alert>
  {:else}
    <form class="flex flex-col gap-4" onsubmit={save} novalidate>
      {#if refill}
        <TextField
          label="Harga isi ulang (Rp)"
          bind:value={refillPrice}
          error={errors.price}
          type="number"
          inputmode="numeric"
          min="0"
          step="500"
          required
        />
      {/if}
      <div class="grid grid-cols-2 gap-4">
        <TextField
          label="Jam buka"
          bind:value={openTime}
          error={errors.open_time}
          type="time"
          required
        />
        <TextField
          label="Jam tutup"
          bind:value={closeTime}
          error={errors.close_time}
          type="time"
          required
        />
      </div>
      <TextField
        label="Ongkos kirim (Rp)"
        bind:value={deliveryFee}
        error={errors.delivery_fee}
        hint="Isi 0 bila gratis."
        type="number"
        inputmode="numeric"
        min="0"
        step="500"
        required
      />
      <TextField
        label="Alamat depot"
        bind:value={address}
        error={errors.address}
        autocomplete="street-address"
      />

      {#if message}
        <Alert kind="error">{message}</Alert>
      {/if}

      <Button type="submit" size="lg" loading={saving}>Simpan dan mulai</Button>
      <a
        href={resolve('/app')}
        class="tap flex items-center justify-center font-semibold text-muted hover:text-text"
      >
        Lewati
      </a>
    </form>
  {/if}
</Card>
