<script lang="ts">
  let {
    label,
    value = $bindable(1),
    min = 0,
    error
  }: { label: string; value?: number; min?: number; error?: string } = $props();

  const id = $props.id();
</script>

<div class="flex flex-col gap-1">
  <label for={id} class="font-medium">{label}</label>
  <div class="flex items-center gap-2">
    <button
      type="button"
      class="tap rounded-lg border border-border bg-surface text-2xl font-bold disabled:opacity-40"
      aria-label="Kurangi {label.toLowerCase()}"
      disabled={value <= min}
      onclick={() => (value = Math.max(min, value - 1))}
    >
      -
    </button>
    <input
      {id}
      type="number"
      inputmode="numeric"
      {min}
      bind:value
      aria-invalid={error ? true : undefined}
      class="min-h-12 w-20 rounded-lg border bg-surface text-center text-xl font-bold text-text {error
        ? 'border-status-cancelled'
        : 'border-border'}"
    />
    <button
      type="button"
      class="tap rounded-lg border border-border bg-surface text-2xl font-bold"
      aria-label="Tambah {label.toLowerCase()}"
      onclick={() => (value = value + 1)}
    >
      +
    </button>
  </div>
  {#if error}
    <p class="text-status-cancelled">{error}</p>
  {/if}
</div>
