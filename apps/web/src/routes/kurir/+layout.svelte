<script lang="ts">
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import Button from '$lib/components/Button.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import { session } from '$lib/state/session.svelte';
  import type { LayoutProps } from './$types';

  let { children }: LayoutProps = $props();

  let loggingOut = $state(false);

  $effect(() => {
    if (session.status === 'unknown') void session.loadSession().catch(() => undefined);
  });

  $effect(() => {
    if (session.status === 'anonymous') void goto(resolve('/masuk'));
    else if (session.user?.role === 'owner') void goto(resolve('/app'));
  });

  async function logout() {
    loggingOut = true;
    try {
      await session.logout();
      await goto(resolve('/masuk'));
    } finally {
      loggingOut = false;
    }
  }

  const ready = $derived(session.status === 'authenticated' && session.user?.role === 'courier');
</script>

{#if ready}
  <div class="flex min-h-dvh flex-col">
    <header
      class="sticky top-0 z-10 flex h-14 items-center justify-between gap-3 border-b border-border bg-surface px-4"
    >
      <div class="min-w-0">
        <p class="truncate text-lg font-bold">{session.user?.name}</p>
        <p class="truncate text-sm text-muted">{session.depot?.name}</p>
      </div>
      <Button variant="ghost" loading={loggingOut} onclick={logout}>Keluar</Button>
    </header>
    <main class="mx-auto w-full max-w-md flex-1 px-4 py-5">
      {@render children()}
    </main>
  </div>
{:else}
  <div class="mx-auto w-full max-w-md px-4 py-8" aria-busy="true">
    <Skeleton lines={6} />
  </div>
{/if}
