<script lang="ts">
  import { resolve } from '$app/paths';
</script>

<svelte:head>
  <title>Daftar | Depotin</title>
</svelte:head>

<main class="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center gap-4 px-4 py-12">
  <h1 class="text-2xl font-bold">Daftar</h1>
  <p class="text-muted">Formulir pendaftaran belum tersedia.</p>
  <a href={resolve('/')} class="tap flex items-center font-semibold text-accent-700"
    >Kembali ke beranda</a
  >
</main>
