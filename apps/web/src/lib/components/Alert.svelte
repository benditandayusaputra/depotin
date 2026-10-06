<script lang="ts">
  import type { Snippet } from 'svelte';

  type Kind = 'error' | 'info' | 'success';

  let {
    kind = 'info',
    onretry,
    children
  }: { kind?: Kind; onretry?: () => void; children: Snippet } = $props();

  const kindClasses: Record<Kind, string> = {
    success: 'border-status-done bg-status-done-soft text-status-done',
    error: 'border-status-cancelled bg-status-cancelled-soft text-status-cancelled',
    info: 'border-status-confirmed bg-status-confirmed-soft text-status-confirmed'
  };
</script>

<div
  role={kind === 'error' ? 'alert' : 'status'}
  class="flex flex-wrap items-center gap-3 rounded-lg border-l-4 px-4 py-3 {kindClasses[kind]}"
>
  <p class="flex-1 font-medium">{@render children()}</p>
  {#if onretry}
    <button type="button" class="tap font-semibold underline" onclick={onretry}>Coba lagi</button>
  {/if}
</div>
