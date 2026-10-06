<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLInputAttributes } from 'svelte/elements';

  interface Props extends Omit<HTMLInputAttributes, 'value' | 'id'> {
    label: string;
    value?: string | number | null;
    error?: string;
    hint?: string;
    trailing?: Snippet;
  }

  let {
    label,
    value = $bindable(''),
    error,
    hint,
    trailing,
    class: className = '',
    ...rest
  }: Props = $props();

  const id = $props.id();
</script>

<div class="flex flex-col gap-1 {className}">
  <label for={id} class="font-medium">{label}</label>
  <div class="relative">
    <input
      {id}
      bind:value
      aria-invalid={error ? true : undefined}
      aria-describedby={error ? `${id}-error` : hint ? `${id}-hint` : undefined}
      class="min-h-12 w-full rounded-lg border bg-surface px-3 text-base text-text placeholder:text-neutral-400 {error
        ? 'border-status-cancelled'
        : 'border-border'} {trailing ? 'pr-14' : ''}"
      {...rest}
    />
    {#if trailing}
      <div class="absolute inset-y-0 right-1 flex items-center">
        {@render trailing()}
      </div>
    {/if}
  </div>
  {#if error}
    <p id="{id}-error" class="text-status-cancelled">{error}</p>
  {:else if hint}
    <p id="{id}-hint" class="text-muted">{hint}</p>
  {/if}
</div>
