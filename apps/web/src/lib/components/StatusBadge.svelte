<script lang="ts">
  import Badge, { type Size, type Tone } from './Badge.svelte';
  import { orderStatusLabel, type OrderStatus } from '$lib/utils/labels';

  let { status, size = 'sm' }: { status: OrderStatus; size?: Size } = $props();

  const tones: Record<OrderStatus, Tone> = {
    pending: 'pending',
    confirmed: 'confirmed',
    on_delivery: 'delivery',
    delivered: 'done',
    cancelled: 'cancelled'
  };
</script>

<Badge tone={tones[status]} {size}>{orderStatusLabel(status)}</Badge>
