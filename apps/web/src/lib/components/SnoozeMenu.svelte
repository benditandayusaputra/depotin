<script lang="ts">
  import Button from '$lib/components/Button.svelte';

  const OPTIONS = [3, 7, 14];

  let { busy = false, onpick }: { busy?: boolean; onpick: (days: number) => void } = $props();

  let open = $state(false);
</script>

<div class="relative">
  <Button
    variant="secondary"
    aria-expanded={open}
    aria-haspopup="menu"
    loading={busy}
    onclick={() => (open = !open)}
  >
    Tunda
  </Button>
  {#if open}
    <ul
      role="menu"
      onkeydown={(event) => event.key === 'Escape' && (open = false)}
      class="absolute right-0 z-20 mt-1 flex min-w-40 flex-col overflow-hidden rounded-lg border border-border bg-surface shadow-card"
    >
      {#each OPTIONS as days (days)}
        <li role="none">
          <button
            type="button"
            role="menuitem"
            class="flex min-h-12 w-full items-center px-4 font-medium hover:bg-neutral-100 dark:hover:bg-neutral-800"
            onclick={() => {
              open = false;
              onpick(days);
            }}
          >
            {days} hari
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>
