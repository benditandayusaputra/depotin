<script lang="ts">
  interface Props {
    label: string;
    value?: number;
    min?: number;
    max?: number;
    class?: string;
  }

  let { label, value = $bindable(1), min = 1, max = 50, class: className = '' }: Props = $props();

  const id = $props.id();
</script>

<div role="group" aria-labelledby={id} class="flex items-center justify-between gap-3 {className}">
  <span {id} class="font-medium">{label}</span>
  <div class="flex items-center gap-2">
    <button
      type="button"
      class="tap rounded-lg border border-border bg-surface text-2xl font-bold disabled:opacity-40"
      aria-label="Kurangi"
      disabled={value <= min}
      onclick={() => (value = Math.max(min, value - 1))}
    >
      &minus;
    </button>
    <output class="w-12 text-center text-2xl font-bold tabular-nums">{value}</output>
    <button
      type="button"
      class="tap rounded-lg border border-border bg-surface text-2xl font-bold disabled:opacity-40"
      aria-label="Tambah"
      disabled={value >= max}
      onclick={() => (value = Math.min(max, value + 1))}
    >
      +
    </button>
  </div>
</div>
