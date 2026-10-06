<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import NavIcon from '$lib/components/NavIcon.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import StreamIndicator from '$lib/components/StreamIndicator.svelte';
  import { ICONS, MORE_NAV, PRIMARY_NAV, SECONDARY_NAV, isActivePath } from '$lib/nav';
  import { session } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { armBeepOnFirstGesture, beep } from '$lib/stream/beep';
  import { stream } from '$lib/stream/client.svelte';
  import { errorMessage } from '$lib/utils/errors';
  import type { LayoutProps } from './$types';

  let { children }: LayoutProps = $props();

  const mobileNav = [...PRIMARY_NAV, MORE_NAV];
  const sidebarNav = [...PRIMARY_NAV, ...SECONDARY_NAV];

  const ORDERS_PATH = '/app/pesanan';
  const FLASH_MS = 1500;

  let loggingOut = $state(false);
  let ordersFlash = $state(false);
  let sessionError = $state('');

  async function loadSession() {
    sessionError = '';
    try {
      await session.loadSession();
    } catch (error) {
      if (session.status === 'unknown') sessionError = errorMessage(error);
    }
  }

  onMount(() => {
    if (session.status === 'unknown') void loadSession();
  });

  onMount(() => armBeepOnFirstGesture());

  $effect(() => {
    if (!ready) return;
    const unsubscribe = untrack(() => {
      stream.connect();
      return stream.subscribe((event) => {
        if (event.type !== 'order.created') return;
        pushToast('info', `Pesanan baru: ${event.data.code}`);
        beep();
        ordersFlash = true;
        setTimeout(() => (ordersFlash = false), FLASH_MS);
      });
    });
    return () => {
      unsubscribe();
      stream.disconnect();
    };
  });

  $effect(() => {
    if (session.status === 'anonymous') void goto(resolve('/masuk'));
    else if (session.user?.role === 'courier') void goto(resolve('/kurir'));
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

  const ready = $derived(session.status === 'authenticated' && session.user?.role === 'owner');
</script>

{#if ready}
  <div class="min-h-dvh md:flex">
    <aside
      class="sticky top-0 hidden h-dvh w-64 shrink-0 flex-col border-r border-border bg-surface md:flex"
    >
      <div class="px-5 py-5">
        <p
          class="text-sm font-semibold tracking-wide text-accent-700 uppercase dark:text-accent-300"
        >
          Depotin
        </p>
        <p class="truncate text-lg font-bold">{session.depot?.name}</p>
        <div class="mt-2"><StreamIndicator /></div>
      </div>
      <nav class="flex flex-1 flex-col gap-1 px-3" aria-label="Menu utama">
        {#each sidebarNav as item (item.href)}
          {@const active = isActivePath(page.url.pathname, item.href)}
          <a
            href={resolve(item.href)}
            aria-current={active ? 'page' : undefined}
            class="flex min-h-12 items-center gap-3 rounded-lg px-3 font-medium transition-colors {active
              ? 'bg-accent-50 text-accent-700 dark:bg-accent-900/40 dark:text-accent-300'
              : 'text-text hover:bg-neutral-100 dark:hover:bg-neutral-800'} {ordersFlash &&
            item.href === ORDERS_PATH
              ? 'ring-2 ring-accent-500'
              : ''}"
          >
            <NavIcon d={item.icon} />
            {item.label}
          </a>
        {/each}
      </nav>
      <div class="p-3">
        <Button variant="ghost" class="w-full" loading={loggingOut} onclick={logout}>
          <NavIcon d={ICONS.logout} class="h-5 w-5" />
          Keluar
        </Button>
      </div>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <header
        class="sticky top-0 z-10 flex h-14 items-center justify-between gap-3 border-b border-border bg-surface px-4 md:hidden"
      >
        <p class="truncate text-lg font-bold">{session.depot?.name}</p>
        <StreamIndicator />
      </header>

      <main class="mx-auto w-full max-w-3xl flex-1 px-4 py-5 pb-24 md:px-8 md:pb-8">
        {@render children()}
      </main>

      <nav
        class="fixed inset-x-0 bottom-0 z-10 grid grid-cols-5 border-t border-border bg-surface md:hidden"
        aria-label="Menu utama"
      >
        {#each mobileNav as item (item.href)}
          {@const active = isActivePath(page.url.pathname, item.href)}
          <a
            href={resolve(item.href)}
            aria-current={active ? 'page' : undefined}
            class="flex min-h-12 flex-col items-center justify-center gap-0.5 py-2 text-sm font-medium transition-colors {active
              ? 'text-accent-700 dark:text-accent-300'
              : 'text-muted'} {ordersFlash && item.href === ORDERS_PATH
              ? 'bg-accent-50 dark:bg-accent-900/40'
              : ''}"
          >
            <NavIcon d={item.icon} />
            {item.label}
          </a>
        {/each}
      </nav>
    </div>
  </div>
{:else if sessionError}
  <div class="mx-auto w-full max-w-3xl px-4 py-8">
    <Alert kind="error" onretry={() => void loadSession()}>{sessionError}</Alert>
  </div>
{:else}
  <div class="mx-auto w-full max-w-3xl px-4 py-8" aria-busy="true">
    <Skeleton lines={6} />
  </div>
{/if}
