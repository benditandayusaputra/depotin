<script lang="ts">
  import { resolve } from '$app/paths';
  import { page } from '$app/state';

  const hint = $derived(
    page.status === 404
      ? 'Periksa kembali link yang Anda terima dari depot.'
      : 'Coba muat ulang beberapa saat lagi.'
  );
</script>

<svelte:head>
  <title>Link tidak dikenal | Depotin</title>
  <meta name="robots" content="noindex" />
</svelte:head>

<main
  class="mx-auto flex min-h-dvh w-full max-w-md flex-col items-center justify-center gap-4 px-4 py-12 text-center"
>
  <p class="text-num text-accent-600">{page.status}</p>
  <h1 class="text-2xl font-bold">{page.error?.message ?? 'Terjadi gangguan.'}</h1>
  <p class="text-muted">{hint}</p>
  <a
    href={resolve('/')}
    class="inline-flex min-h-12 items-center rounded-lg bg-accent-600 px-6 font-semibold text-white"
  >
    Ke beranda
  </a>
</main>
