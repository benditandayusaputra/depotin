<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    selected = false,
    onclick,
    children
  }: { selected?: boolean; onclick: () => void; children: Snippet } = $props();
</script>

<button
  type="button"
  aria-pressed={selected}
  class="min-h-11 shrink-0 rounded-full border px-4 font-medium {selected
    ? 'border-accent-600 bg-accent-600 text-white'
    : 'border-border bg-surface text-text hover:bg-neutral-100 dark:hover:bg-neutral-800'}"
  {onclick}
>
  {@render children()}
</button>
