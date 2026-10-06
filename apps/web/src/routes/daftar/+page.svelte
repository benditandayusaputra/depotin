<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import PasswordField from '$lib/components/PasswordField.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { session, type SessionData } from '$lib/state/session.svelte';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';

  let depotName = $state('');
  let name = $state('');
  let phone = $state('');
  let password = $state('');
  let submitting = $state(false);
  let hydrated = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    submitting = true;
    errors = {};
    message = '';
    try {
      const data = await apiFetch<SessionData>('/auth/register', {
        body: { depot_name: depotName, name, phone, password }
      });
      session.setSession(data);
      await goto(resolve('/app/penyiapan'));
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      submitting = false;
    }
  }

  onMount(() => {
    hydrated = true;
  });
</script>

<svelte:head>
  <title>Daftar | Depotin</title>
</svelte:head>

<main class="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center gap-6 px-4 py-12">
  <div>
    <h1 class="text-2xl font-bold">Daftar depot</h1>
    <p class="mt-1 text-muted">Gratis, cukup satu menit.</p>
  </div>

  <form class="flex flex-col gap-4" onsubmit={submit} novalidate>
    <TextField label="Nama depot" bind:value={depotName} error={errors.depot_name} required />
    <TextField
      label="Nama Anda"
      bind:value={name}
      error={errors.name}
      autocomplete="name"
      required
    />
    <TextField
      label="Nomor HP"
      bind:value={phone}
      error={errors.phone}
      type="tel"
      inputmode="tel"
      autocomplete="tel"
      placeholder="08xxxxxxxxxx"
      required
    />
    <PasswordField
      bind:value={password}
      error={errors.password}
      hint="Minimal 10 karakter."
      autocomplete="new-password"
      minlength={10}
    />

    {#if message}
      <Alert kind="error">{message}</Alert>
    {/if}

    <Button type="submit" size="lg" loading={submitting} disabled={!hydrated}>Daftar</Button>
  </form>

  <p class="text-center text-muted">
    Sudah punya akun?
    <a href={resolve('/masuk')} class="font-semibold text-accent-700 dark:text-accent-300">Masuk</a>
  </p>
</main>
