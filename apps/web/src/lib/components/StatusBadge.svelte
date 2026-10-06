<script lang="ts">
  import { statusClasses, statusLabel, type OrderStatus } from '$lib/utils/status';

  let { status, class: className = '' }: { status: OrderStatus; class?: string } = $props();
</script>

<span
  class="inline-flex shrink-0 items-center rounded-md px-2.5 py-1 font-semibold {statusClasses(
    status
  )} {className}"
>
  {statusLabel(status)}
</span>
