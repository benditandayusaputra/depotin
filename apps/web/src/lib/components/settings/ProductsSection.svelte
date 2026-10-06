<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import type { components } from '$lib/api/schema';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import ProductRow from '$lib/components/settings/ProductRow.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';
  import { PRODUCT_KINDS, productKindLabel, type ProductKind } from '$lib/utils/labels';

  type Product = components['schemas']['Product'];

  let products = $state<Product[]>([]);
  let loading = $state(true);
  let loadError = $state('');

  let name = $state('');
  let kind = $state<ProductKind>('refill');
  let price = $state('');
  let adding = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');

  async function load() {
    loading = true;
    loadError = '';
    try {
      products = await apiFetch<Product[]>('/products');
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  function replaceProduct(updated: Product) {
    products = products.map((product) => (product.id === updated.id ? updated : product));
  }

  async function add(event: SubmitEvent) {
    event.preventDefault();
    adding = true;
    errors = {};
    message = '';
    try {
      const created = await apiFetch<Product>('/products', {
        body: { name, kind, price: Number(price) }
      });
      products = [...products, created];
      name = '';
      price = '';
      pushToast('success', 'Produk ditambahkan.');
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      adding = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<div class="flex flex-col gap-4">
  <Card>
    <h2 class="mb-3 text-lg font-semibold">Daftar produk</h2>
    {#if loading}
      <Skeleton lines={4} />
    {:else if loadError}
      <Alert kind="error" onretry={load}>{loadError}</Alert>
    {:else if products.length === 0}
      <EmptyState
        title="Belum ada produk"
        description="Tambahkan produk pertama lewat formulir di bawah."
      />
    {:else}
      <ul class="divide-y divide-border">
        {#each products as product (product.id)}
          <ProductRow {product} onchanged={replaceProduct} />
        {/each}
      </ul>
    {/if}
  </Card>

  <Card>
    <h2 class="mb-3 text-lg font-semibold">Tambah produk</h2>
    <form class="flex flex-col gap-4" onsubmit={add} novalidate>
      <TextField label="Nama produk" bind:value={name} error={errors.name} required />
      <div class="flex flex-col gap-1">
        <label for="product-kind" class="font-medium">Jenis</label>
        <select
          id="product-kind"
          bind:value={kind}
          class="min-h-12 rounded-lg border border-border bg-surface px-3 text-base"
        >
          {#each PRODUCT_KINDS as option (option)}
            <option value={option}>{productKindLabel(option)}</option>
          {/each}
        </select>
        {#if errors.kind}
          <p class="text-status-cancelled">{errors.kind}</p>
        {/if}
      </div>
      <TextField
        label="Harga (Rp)"
        bind:value={price}
        error={errors.price}
        type="number"
        inputmode="numeric"
        min="0"
        step="500"
        required
      />

      {#if message}
        <Alert kind="error">{message}</Alert>
      {/if}

      <Button type="submit" size="lg" loading={adding}>Tambah produk</Button>
    </form>
  </Card>
</div>
