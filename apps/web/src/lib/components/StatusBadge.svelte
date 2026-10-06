<script lang="ts">
  import Badge, { type Tone } from './Badge.svelte';
  import { orderStatusLabel, type OrderStatus } from '$lib/utils/labels';

  let { status }: { status: OrderStatus } = $props();

  const tones: Record<OrderStatus, Tone> = {
    pending: 'pending',
    confirmed: 'confirmed',
    on_delivery: 'delivery',
    delivered: 'done',
    cancelled: 'cancelled'
  };
</script>

<Badge tone={tones[status]}>{orderStatusLabel(status)}</Badge>
