<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    label,
    value = $bindable(''),
    error,
    disabled = false,
    children
  }: {
    label: string;
    value?: string;
    error?: string;
    disabled?: boolean;
    children: Snippet;
  } = $props();

  const id = $props.id();
</script>

<div class="flex flex-col gap-1">
  <label for={id} class="font-medium">{label}</label>
  <select
    {id}
    bind:value
    {disabled}
    aria-invalid={error ? true : undefined}
    class="min-h-12 w-full rounded-lg border bg-surface px-3 text-base text-text {error
      ? 'border-status-cancelled'
      : 'border-border'}"
  >
    {@render children()}
  </select>
  {#if error}
    <p class="text-status-cancelled">{error}</p>
  {/if}
</div>
