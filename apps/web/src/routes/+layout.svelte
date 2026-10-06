<script lang="ts">
  import '../app.css';
  import Toaster from '$lib/components/Toaster.svelte';
  import type { LayoutProps } from './$types';

  let { children }: LayoutProps = $props();
</script>

{@render children()}
<Toaster />
