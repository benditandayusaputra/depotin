<script lang="ts">
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import PasswordField from '$lib/components/PasswordField.svelte';
  import { session } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';

  let currentPassword = $state('');
  let newPassword = $state('');
  let saving = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');
  let loggingOutAll = $state(false);
  let logoutMessage = $state('');

  async function changePassword(event: SubmitEvent) {
    event.preventDefault();
    saving = true;
    errors = {};
    message = '';
    try {
      await apiFetch('/auth/password', {
        body: { current_password: currentPassword, new_password: newPassword }
      });
      currentPassword = '';
      newPassword = '';
      pushToast('success', 'Kata sandi berhasil diganti.');
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      saving = false;
    }
  }

  async function logoutAll() {
    loggingOutAll = true;
    logoutMessage = '';
    try {
      await apiFetch('/auth/logout-all', { method: 'POST' });
      session.clearSession();
      await goto(resolve('/masuk'));
    } catch (error) {
      logoutMessage = errorMessage(error);
    } finally {
      loggingOutAll = false;
    }
  }
</script>

<div class="flex flex-col gap-4">
  <Card>
    <h2 class="mb-3 text-lg font-semibold">Ganti kata sandi</h2>
    <form class="flex flex-col gap-4" onsubmit={changePassword} novalidate>
      <PasswordField
        label="Kata sandi sekarang"
        bind:value={currentPassword}
        error={errors.current_password}
        autocomplete="current-password"
      />
      <PasswordField
        label="Kata sandi baru"
        bind:value={newPassword}
        error={errors.new_password}
        hint="Minimal 10 karakter."
        autocomplete="new-password"
        minlength={10}
      />

      {#if message}
        <Alert kind="error">{message}</Alert>
      {/if}

      <Button type="submit" size="lg" loading={saving}>Ganti kata sandi</Button>
    </form>
  </Card>

  <Card>
    <h2 class="text-lg font-semibold">Keluar dari semua perangkat</h2>
    <p class="mt-1 mb-4 text-muted">Semua sesi aktif, termasuk di perangkat ini, akan diakhiri.</p>
    {#if logoutMessage}
      <div class="mb-4">
        <Alert kind="error" onretry={logoutAll}>{logoutMessage}</Alert>
      </div>
    {/if}
    <Button variant="danger" size="lg" loading={loggingOutAll} onclick={logoutAll}>
      Keluar dari semua perangkat
    </Button>
  </Card>
</div>
