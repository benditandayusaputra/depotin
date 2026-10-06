<script lang="ts">
  import type { Snippet } from 'svelte';

  export type Tone = 'pending' | 'confirmed' | 'delivery' | 'done' | 'cancelled' | 'neutral';

  export type Size = 'sm' | 'md';

  let {
    tone = 'neutral',
    size = 'sm',
    children
  }: { tone?: Tone; size?: Size; children: Snippet } = $props();

  const toneClasses: Record<Tone, string> = {
    pending: 'bg-status-pending-soft text-status-pending',
    confirmed: 'bg-status-confirmed-soft text-status-confirmed',
    delivery: 'bg-status-delivery-soft text-status-delivery',
    done: 'bg-status-done-soft text-status-done',
    cancelled: 'bg-status-cancelled-soft text-status-cancelled',
    neutral: 'bg-neutral-200 text-neutral-700 dark:bg-neutral-700 dark:text-neutral-200'
  };
</script>

<span
  class="inline-flex items-center rounded-full px-2.5 py-0.5 font-semibold whitespace-nowrap {size ===
  'sm'
    ? 'text-sm'
    : 'text-base'} {toneClasses[tone]}"
>
  {@render children()}
</span>
