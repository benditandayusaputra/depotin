<script lang="ts">
  import AccountSection from '$lib/components/settings/AccountSection.svelte';
  import CouriersSection from '$lib/components/settings/CouriersSection.svelte';
  import DepotProfileSection from '$lib/components/settings/DepotProfileSection.svelte';
  import LoyaltySection from '$lib/components/settings/LoyaltySection.svelte';
  import ProductsSection from '$lib/components/settings/ProductsSection.svelte';
  import RemindersSection from '$lib/components/settings/RemindersSection.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import { session } from '$lib/state/session.svelte';

  const TABS = [
    { id: 'profil', label: 'Profil depot' },
    { id: 'produk', label: 'Produk dan harga' },
    { id: 'kurir', label: 'Kurir' },
    { id: 'loyalitas', label: 'Loyalitas' },
    { id: 'pengingat', label: 'Pengingat' },
    { id: 'akun', label: 'Akun' }
  ] as const;

  type TabId = (typeof TABS)[number]['id'];

  let activeTab = $state<TabId>('profil');
</script>

<svelte:head>
  <title>Pengaturan | Depotin</title>
</svelte:head>

<PageHeader title="Pengaturan" />

<div
  role="tablist"
  aria-label="Bagian pengaturan"
  class="-mx-4 mb-4 flex gap-2 overflow-x-auto px-4 pb-1 md:mx-0 md:flex-wrap md:px-0"
>
  {#each TABS as tab (tab.id)}
    <button
      type="button"
      role="tab"
      id="tab-{tab.id}"
      aria-selected={activeTab === tab.id}
      aria-controls="panel-{tab.id}"
      class="min-h-11 shrink-0 rounded-full border px-4 font-medium {activeTab === tab.id
        ? 'border-accent-600 bg-accent-600 text-white'
        : 'border-border bg-surface text-text hover:bg-neutral-100 dark:hover:bg-neutral-800'}"
      onclick={() => (activeTab = tab.id)}
    >
      {tab.label}
    </button>
  {/each}
</div>

<div role="tabpanel" id="panel-{activeTab}" aria-labelledby="tab-{activeTab}">
  {#if !session.depot}
    <Skeleton lines={6} />
  {:else if activeTab === 'profil'}
    <DepotProfileSection depot={session.depot} />
  {:else if activeTab === 'produk'}
    <ProductsSection />
  {:else if activeTab === 'kurir'}
    <CouriersSection />
  {:else if activeTab === 'loyalitas'}
    <LoyaltySection depot={session.depot} />
  {:else if activeTab === 'pengingat'}
    <RemindersSection depot={session.depot} />
  {:else}
    <AccountSection />
  {/if}
</div>
