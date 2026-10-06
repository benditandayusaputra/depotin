<script lang="ts">
  import { untrack } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import type { components } from '$lib/api/schema';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import Toggle from '$lib/components/Toggle.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage, fieldErrors } from '$lib/utils/errors';
  import { productKindLabel } from '$lib/utils/labels';

  type Product = components['schemas']['Product'];
  type ProductUpdate = components['schemas']['ProductUpdate'];

  let { product, onchanged }: { product: Product; onchanged: (product: Product) => void } =
    $props();

  let price = $state(untrack(() => String(product.price)));
  let active = $state(untrack(() => product.is_active));
  let saving = $state(false);
  let priceError = $state('');
  let message = $state('');

  const priceChanged = $derived(Number(price) !== product.price);

  async function update(body: ProductUpdate, successMessage: string) {
    saving = true;
    priceError = '';
    message = '';
    try {
      onchanged(await apiFetch<Product>(`/products/${product.id}`, { method: 'PATCH', body }));
      pushToast('success', successMessage);
    } catch (error) {
      priceError = fieldErrors(error).price ?? '';
      if (!priceError) message = errorMessage(error);
      active = product.is_active;
    } finally {
      saving = false;
    }
  }

  function savePrice(event: SubmitEvent) {
    event.preventDefault();
    void update({ price: Number(price) }, 'Harga tersimpan.');
  }

  function toggleActive(checked: boolean) {
    void update({ is_active: checked }, checked ? 'Produk diaktifkan.' : 'Produk dinonaktifkan.');
  }
</script>

<li class="flex flex-col gap-3 py-4">
  <div class="flex items-start justify-between gap-3">
    <div>
      <p class="font-semibold">{product.name}</p>
      <p class="text-muted">{productKindLabel(product.kind)}</p>
    </div>
    <Toggle label="Aktif" bind:checked={active} disabled={saving} onchange={toggleActive} />
  </div>
  <form class="flex items-end gap-2" onsubmit={savePrice}>
    <TextField
      label="Harga {product.name} (Rp)"
      bind:value={price}
      error={priceError}
      type="number"
      inputmode="numeric"
      min="0"
      step="500"
      class="flex-1"
      required
    />
    <Button type="submit" variant="secondary" size="lg" loading={saving} disabled={!priceChanged}>
      Simpan
    </Button>
  </form>
  {#if message}
    <Alert kind="error">{message}</Alert>
  {/if}
</li>
