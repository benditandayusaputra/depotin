<script lang="ts">
  import { resolve } from '$app/paths';
</script>

<svelte:head>
  <title>Depotin</title>
  <meta
    name="description"
    content="Aplikasi depot air isi ulang untuk pesanan, pengingat, dan antar galon."
  />
</svelte:head>

<main class="flex min-h-dvh flex-col items-center justify-center px-4 py-12 text-center">
  <div class="flex w-full max-w-md flex-col items-center gap-6">
    <svg
      class="h-16 w-16 text-accent-600"
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M12 2.5c-3.6 4.6-7 8.6-7 12.3a7 7 0 0 0 14 0c0-3.7-3.4-7.7-7-12.3Z" />
    </svg>
    <h1 class="text-3xl font-bold tracking-tight">Depotin</h1>
    <p class="text-lg text-muted">
      Catat pesanan, kirim pengingat, dan antar galon dari HP tanpa ribet.
    </p>
    <div class="flex w-full flex-col gap-3 sm:flex-row">
      <a
        href={resolve('/masuk')}
        class="tap flex flex-1 items-center justify-center rounded-lg bg-accent-600 px-6 font-semibold text-white hover:bg-accent-700"
      >
        Masuk
      </a>
      <a
        href={resolve('/daftar')}
        class="tap flex flex-1 items-center justify-center rounded-lg border border-border bg-surface px-6 font-semibold text-accent-700 hover:bg-accent-50 dark:text-accent-300 dark:hover:bg-accent-900/40"
      >
        Daftar
      </a>
    </div>
  </div>
</main>
