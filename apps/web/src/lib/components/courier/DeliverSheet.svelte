<script lang="ts" module>
  import type { components } from '$lib/api/schema';

  export interface DeliverPayload {
    gallons_returned: number;
    payment_method?: components['schemas']['PaymentMethod'];
    paid?: boolean;
  }
</script>

<script lang="ts">
  import Button from '$lib/components/Button.svelte';
  import CounterStepper from '$lib/components/CounterStepper.svelte';
  import { formatRupiah } from '$lib/utils/rupiah';

  type CourierOrder = components['schemas']['CourierOrder'];
  type Payment = 'cash' | 'transfer' | 'unpaid';

  interface Props {
    order: CourierOrder;
    onsubmit: (payload: DeliverPayload, saveLocation: boolean) => void;
    onclose: () => void;
  }

  let { order, onsubmit, onclose }: Props = $props();

  const PAYMENT_OPTIONS: { value: Payment; label: string }[] = [
    { value: 'cash', label: 'Tunai' },
    { value: 'transfer', label: 'Transfer' },
    { value: 'unpaid', label: 'Belum bayar' }
  ];

  const titleId = $props.id();

  let returned = $derived(order.refill_qty);
  let payment = $state<Payment | null>(null);
  let saveLocation = $state(false);

  const alreadyPaid = $derived(order.payment_status === 'paid');
  const ready = $derived(alreadyPaid || payment !== null);

  function submit() {
    const payload: DeliverPayload = { gallons_returned: returned };
    if (!alreadyPaid) {
      payload.paid = payment !== 'unpaid';
      if (payment === 'cash' || payment === 'transfer') payload.payment_method = payment;
    }
    onsubmit(payload, saveLocation);
  }
</script>

<svelte:window onkeydown={(event) => event.key === 'Escape' && onclose()} />

<button
  type="button"
  class="fixed inset-0 z-40 bg-neutral-900/50"
  aria-label="Tutup"
  onclick={onclose}
></button>

<div
  role="dialog"
  aria-modal="true"
  aria-labelledby={titleId}
  class="fixed inset-x-0 bottom-0 z-50 mx-auto flex w-full max-w-md flex-col gap-4 rounded-t-xl bg-surface p-4 pb-6 shadow-card"
>
  <div>
    <h2 id={titleId} class="text-xl font-bold">Selesaikan {order.delivery_name}</h2>
    <p class="text-muted">
      {order.refill_qty} galon.
      {#if alreadyPaid}Sudah lunas.{:else}Tagih {formatRupiah(order.total)}.{/if}
    </p>
  </div>

  <CounterStepper
    label="Galon kosong diterima"
    bind:value={returned}
    min={0}
    max={order.refill_qty + order.loan_balance}
  />

  {#if !alreadyPaid}
    <div class="grid grid-cols-3 gap-2" role="group" aria-label="Cara bayar">
      {#each PAYMENT_OPTIONS as option (option.value)}
        <button
          type="button"
          aria-pressed={payment === option.value}
          class="min-h-14 rounded-lg border-2 px-2 font-semibold {payment === option.value
            ? 'border-accent-600 bg-accent-50 text-accent-700 dark:bg-accent-900/40 dark:text-accent-300'
            : 'border-border bg-surface'}"
          onclick={() => (payment = option.value)}
        >
          {option.label}
        </button>
      {/each}
    </div>
  {/if}

  <label class="flex min-h-12 items-center gap-3">
    <input type="checkbox" bind:checked={saveLocation} class="h-6 w-6 accent-accent-600" />
    <span>Simpan lokasi saya di sini</span>
  </label>

  <Button size="lg" class="min-h-16 text-xl" disabled={!ready} onclick={submit}>Selesai</Button>
  <Button variant="ghost" size="lg" onclick={onclose}>Batal</Button>
</div>
