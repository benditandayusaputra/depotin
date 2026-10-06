<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  type Variant = 'primary' | 'secondary' | 'danger' | 'ghost';
  type Size = 'md' | 'lg';

  interface Props extends HTMLButtonAttributes {
    variant?: Variant;
    size?: Size;
    loading?: boolean;
    children: Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    loading = false,
    disabled = false,
    type = 'button',
    class: className = '',
    children,
    ...rest
  }: Props = $props();

  const variantClasses: Record<Variant, string> = {
    primary: 'bg-accent-600 text-white hover:bg-accent-700',
    secondary:
      'border border-border bg-surface text-accent-700 hover:bg-accent-50 dark:text-accent-300 dark:hover:bg-accent-900/40',
    danger: 'bg-status-cancelled text-white hover:opacity-90 dark:text-neutral-900',
    ghost: 'text-accent-700 hover:bg-accent-50 dark:text-accent-300 dark:hover:bg-accent-900/40'
  };

  const sizeClasses: Record<Size, string> = {
    md: 'min-h-11 px-4',
    lg: 'min-h-12 px-6 text-lg'
  };
</script>

<button
  {type}
  disabled={disabled || loading}
  aria-busy={loading}
  class="inline-flex items-center justify-center gap-2 rounded-lg font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-60 {variantClasses[
    variant
  ]} {sizeClasses[size]} {className}"
  {...rest}
>
  {#if loading}
    <span
      class="h-5 w-5 shrink-0 animate-spin rounded-full border-2 border-current border-t-transparent"
      aria-hidden="true"
    ></span>
  {/if}
  {@render children()}
</button>
