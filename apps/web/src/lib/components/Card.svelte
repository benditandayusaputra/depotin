<script lang="ts">
  import type { Snippet } from 'svelte';

  let { children, class: className = '' }: { children: Snippet; class?: string } = $props();
</script>

<section class="rounded-lg border border-border bg-surface p-4 shadow-card {className}">
  {@render children()}
</section>
