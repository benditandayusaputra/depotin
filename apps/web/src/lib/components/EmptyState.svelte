<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    description,
    icon,
    action
  }: { title: string; description?: string; icon?: Snippet; action?: Snippet } = $props();
</script>

<div
  class="flex flex-col items-center gap-3 rounded-lg border border-dashed border-border px-4 py-10 text-center"
>
  {#if icon}
    <div class="text-accent-600">{@render icon()}</div>
  {/if}
  <p class="text-lg font-semibold">{title}</p>
  {#if description}
    <p class="max-w-sm text-muted">{description}</p>
  {/if}
  {#if action}
    <div class="mt-2">{@render action()}</div>
  {/if}
</div>
