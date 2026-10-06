<script lang="ts">
  import TextField from './TextField.svelte';

  interface Props {
    label?: string;
    value?: string;
    error?: string;
    hint?: string;
    autocomplete?: 'current-password' | 'new-password';
    required?: boolean;
    minlength?: number;
    disabled?: boolean;
  }

  let {
    label = 'Kata sandi',
    value = $bindable(''),
    error,
    hint,
    autocomplete = 'current-password',
    required = true,
    minlength,
    disabled = false
  }: Props = $props();

  let visible = $state(false);
</script>

<TextField
  {label}
  bind:value
  {error}
  {hint}
  {autocomplete}
  {required}
  {minlength}
  {disabled}
  type={visible ? 'text' : 'password'}
>
  {#snippet trailing()}
    <button
      type="button"
      class="tap rounded-md px-2 text-sm font-semibold text-accent-700 dark:text-accent-300"
      aria-pressed={visible}
      onclick={() => (visible = !visible)}
    >
      {visible ? 'Sembunyikan' : 'Tampilkan'}
    </button>
  {/snippet}
</TextField>
