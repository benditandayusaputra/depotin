<script lang="ts">
  import type { Snippet } from 'svelte';

  let { title, description, actions }: { title: string; description?: string; actions?: Snippet } =
    $props();
</script>

<header class="mb-4 flex flex-wrap items-start justify-between gap-3">
  <div>
    <h1 class="text-2xl font-bold tracking-tight">{title}</h1>
    {#if description}
      <p class="mt-1 text-muted">{description}</p>
    {/if}
  </div>
  {#if actions}
    <div class="flex shrink-0 gap-2">{@render actions()}</div>
  {/if}
</header>
