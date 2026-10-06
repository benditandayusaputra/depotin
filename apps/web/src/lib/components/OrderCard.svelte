<script lang="ts">
  import type { Snippet } from 'svelte';
  import { resolve } from '$app/paths';
  import Badge from '$lib/components/Badge.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import type { Order } from '$lib/api/types';
  import { formatTime } from '$lib/utils/date';
  import { fulfilmentLabel, orderSourceLabel } from '$lib/utils/labels';
  import { formatRupiah } from '$lib/utils/rupiah';

  let {
    order,
    showStatus = true,
    actions
  }: { order: Order; showStatus?: boolean; actions?: Snippet } = $props();
</script>

<li class="flex flex-col gap-2 rounded-lg border border-border bg-surface p-3 shadow-card">
  <a
    href={resolve('/app/pesanan/[id]', { id: order.id })}
    class="flex min-w-0 flex-col gap-1 rounded-md focus-ring"
    aria-label="Buka pesanan {order.code}"
  >
    <div class="flex items-start justify-between gap-2">
      <div class="min-w-0">
        <p class="truncate font-semibold">{order.delivery_name}</p>
        <p class="text-sm text-muted">{order.code} &middot; {formatTime(order.created_at)}</p>
      </div>
      <p class="shrink-0 text-lg font-bold">{formatRupiah(order.total)}</p>
    </div>
    <div class="flex flex-wrap items-center gap-1.5">
      {#if showStatus}
        <StatusBadge status={order.status} />
      {/if}
      <Badge>{order.refill_qty} galon</Badge>
      <Badge>{fulfilmentLabel(order.fulfilment)}</Badge>
      <Badge>{orderSourceLabel(order.source)}</Badge>
      {#if order.courier_name}
        <Badge tone="delivery">Kurir: {order.courier_name}</Badge>
      {/if}
    </div>
  </a>
  {#if actions}
    <div class="flex flex-wrap gap-2">{@render actions()}</div>
  {/if}
</li>
