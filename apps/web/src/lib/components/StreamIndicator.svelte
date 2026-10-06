<script lang="ts">
  import { stream, type StreamStatus } from '$lib/stream/client.svelte';

  const labels: Record<StreamStatus, string> = {
    connected: 'Tersambung',
    connecting: 'Menyambung',
    reconnecting: 'Menyambung ulang',
    polling: 'Mode berkala',
    off: 'Tidak tersambung'
  };

  const dotClasses: Record<StreamStatus, string> = {
    connected: 'bg-status-done',
    connecting: 'bg-status-pending animate-pulse',
    reconnecting: 'bg-status-pending animate-pulse',
    polling: 'bg-status-pending',
    off: 'bg-neutral-400'
  };
</script>

<p class="flex items-center gap-2 text-sm text-muted" role="status" aria-live="polite">
  <span class="h-2.5 w-2.5 shrink-0 rounded-full {dotClasses[stream.status]}" aria-hidden="true"
  ></span>
  <span>{labels[stream.status]}</span>
</p>
