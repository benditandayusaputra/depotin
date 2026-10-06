<script lang="ts">
  import EmptyState from './EmptyState.svelte';
  import PageHeader from './PageHeader.svelte';

  let { title }: { title: string } = $props();
</script>

<svelte:head>
  <title>{title} | Depotin</title>
</svelte:head>

<PageHeader {title} />
<EmptyState title="Halaman ini sedang disiapkan." />
