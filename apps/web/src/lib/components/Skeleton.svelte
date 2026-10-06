<script lang="ts">
  let { lines = 3, class: className = '' }: { lines?: number; class?: string } = $props();
</script>

<div class="flex animate-pulse flex-col gap-3 {className}" aria-hidden="true">
  {#each { length: lines }, index (index)}
    <div
      class="h-4 rounded bg-neutral-200 dark:bg-neutral-700 {index % 3 === 2 ? 'w-2/3' : ''}"
    ></div>
  {/each}
</div>
