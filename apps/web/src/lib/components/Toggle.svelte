<script lang="ts">
  interface Props {
    label: string;
    checked?: boolean;
    disabled?: boolean;
    onchange?: (checked: boolean) => void;
  }

  let { label, checked = $bindable(false), disabled = false, onchange }: Props = $props();
</script>

<label class="flex min-h-12 cursor-pointer items-center justify-between gap-4">
  <span class="font-medium">{label}</span>
  <input
    type="checkbox"
    role="switch"
    bind:checked
    {disabled}
    onchange={() => onchange?.(checked)}
    class="h-6 w-6 shrink-0 accent-accent-600 disabled:opacity-50"
  />
</label>
