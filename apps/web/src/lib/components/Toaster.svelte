<script lang="ts">
  import { dismissToast, toasts, type ToastKind } from '$lib/state/toast.svelte';

  const kindClasses: Record<ToastKind, string> = {
    success: 'border-status-done bg-status-done-soft text-status-done',
    error: 'border-status-cancelled bg-status-cancelled-soft text-status-cancelled',
    info: 'border-status-confirmed bg-status-confirmed-soft text-status-confirmed'
  };
</script>

<div
  aria-live="polite"
  class="pointer-events-none fixed inset-x-0 bottom-20 z-50 md:bottom-4 flex flex-col items-center gap-2 px-4"
>
  {#each toasts as toast (toast.id)}
    <div
      role="status"
      class="pointer-events-auto flex w-full max-w-md items-center gap-3 rounded-lg border-l-4 bg-surface py-2 pr-1 pl-4 shadow-card {kindClasses[
        toast.kind
      ]}"
    >
      <span class="flex-1 font-medium">{toast.message}</span>
      <button
        type="button"
        class="tap flex items-center justify-center rounded-md text-xl"
        aria-label="Tutup"
        onclick={() => dismissToast(toast.id)}
      >
        &times;
      </button>
    </div>
  {/each}
</div>
