<script lang="ts">
  import { onMount } from 'svelte';
  import { env } from '$env/dynamic/public';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import PasswordField from '$lib/components/PasswordField.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { session, type SessionData } from '$lib/state/session.svelte';
  import { errorMessage } from '$lib/utils/errors';

  const DEMO_PASSWORD = 'demo-depotin-2026';
  const DEMO_ACCOUNTS = [
    { label: 'Pemilik', phone: '081200000001' },
    { label: 'Kurir', phone: '081200000002' }
  ];
  const demoMode = env.PUBLIC_DEMO_MODE === 'true';

  let phone = $state('');
  let password = $state('');
  let submitting = $state(false);
  let hydrated = $state(false);
  let message = $state('');

  function fillDemo(demoPhone: string) {
    phone = demoPhone;
    password = DEMO_PASSWORD;
    message = '';
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    submitting = true;
    message = '';
    try {
      const data = await apiFetch<SessionData>('/auth/login', { body: { phone, password } });
      session.setSession(data);
      await goto(resolve(data.user.role === 'owner' ? '/app' : '/kurir'));
    } catch (error) {
      message = errorMessage(error);
    } finally {
      submitting = false;
    }
  }

  onMount(() => {
    hydrated = true;
  });
</script>

<svelte:head>
  <title>Masuk | Depotin</title>
</svelte:head>

<main class="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center gap-6 px-4 py-12">
  <div>
    <h1 class="text-2xl font-bold">Masuk</h1>
    <p class="mt-1 text-muted">Pakai nomor HP yang terdaftar.</p>
  </div>

  <form class="flex flex-col gap-4" onsubmit={submit} novalidate>
    <TextField
      label="Nomor HP"
      bind:value={phone}
      type="tel"
      inputmode="tel"
      autocomplete="tel"
      placeholder="08xxxxxxxxxx"
      required
    />
    <PasswordField bind:value={password} autocomplete="current-password" />

    {#if message}
      <Alert kind="error">{message}</Alert>
    {/if}

    <Button type="submit" size="lg" loading={submitting} disabled={!hydrated}>Masuk</Button>
  </form>

  {#if demoMode}
    <Card>
      <h2 class="font-semibold">Akun demo</h2>
      <p class="mt-1 text-muted">Kata sandi: {DEMO_PASSWORD}</p>
      <ul class="mt-3 flex flex-col divide-y divide-border">
        {#each DEMO_ACCOUNTS as account (account.phone)}
          <li class="flex items-center justify-between gap-3 py-2">
            <span>
              <span class="font-medium">{account.label}</span>
              <span class="block text-muted">{account.phone}</span>
            </span>
            <Button variant="secondary" onclick={() => fillDemo(account.phone)}>Isi formulir</Button
            >
          </li>
        {/each}
      </ul>
    </Card>
  {/if}

  <p class="text-center text-muted">
    Belum punya akun?
    <a
      href={resolve('/daftar')}
      class="inline-flex min-h-11 items-center px-1 font-semibold text-accent-700 dark:text-accent-300"
      >Daftar</a
    >
  </p>
</main>
