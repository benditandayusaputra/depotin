<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAnchorAttributes } from 'svelte/elements';

  type Variant = 'primary' | 'secondary';

  interface Props extends HTMLAnchorAttributes {
    variant?: Variant;
    external?: boolean;
    children: Snippet;
  }

  let {
    variant = 'primary',
    class: className = '',
    external = false,
    children,
    ...rest
  }: Props = $props();

  const variantClasses: Record<Variant, string> = {
    primary: 'bg-accent-600 text-white hover:bg-accent-700',
    secondary:
      'border border-border bg-surface text-accent-700 hover:bg-accent-50 dark:text-accent-300 dark:hover:bg-accent-900/40'
  };
</script>

<a
  {...rest}
  target={external ? '_blank' : undefined}
  rel={external ? 'noopener' : undefined}
  class="inline-flex min-h-12 items-center justify-center gap-2 rounded-lg px-4 font-semibold {variantClasses[
    variant
  ]} {className}"
>
  {@render children()}
</a>
