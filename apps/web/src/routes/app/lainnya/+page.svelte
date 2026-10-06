<script lang="ts">
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import NavIcon from '$lib/components/NavIcon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { ICONS, SECONDARY_NAV } from '$lib/nav';
  import { session } from '$lib/state/session.svelte';

  let loggingOut = $state(false);

  async function logout() {
    loggingOut = true;
    try {
      await session.logout();
      await goto(resolve('/masuk'));
    } finally {
      loggingOut = false;
    }
  }
</script>

<svelte:head>
  <title>Lainnya | Depotin</title>
</svelte:head>

<PageHeader title="Lainnya" />

<ul class="divide-y divide-border overflow-hidden rounded-lg border border-border bg-surface">
  {#each SECONDARY_NAV as item (item.href)}
    <li>
      <a
        href={resolve(item.href)}
        class="flex min-h-14 items-center gap-4 px-4 text-lg font-medium hover:bg-neutral-100 dark:hover:bg-neutral-800"
      >
        <NavIcon d={item.icon} class="h-6 w-6 text-accent-600" />
        {item.label}
      </a>
    </li>
  {/each}
  <li>
    <button
      type="button"
      class="flex min-h-14 w-full items-center gap-4 px-4 text-lg font-medium text-status-cancelled hover:bg-neutral-100 disabled:opacity-60 dark:hover:bg-neutral-800"
      disabled={loggingOut}
      onclick={logout}
    >
      <NavIcon d={ICONS.logout} class="h-6 w-6" />
      {loggingOut ? 'Keluar...' : 'Keluar'}
    </button>
  </li>
</ul>
