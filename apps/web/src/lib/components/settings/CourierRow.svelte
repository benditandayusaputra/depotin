<script lang="ts">
  import { untrack } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import PasswordField from '$lib/components/PasswordField.svelte';
  import Toggle from '$lib/components/Toggle.svelte';
  import type { User } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage, fieldErrors } from '$lib/utils/errors';

  let { courier, onchanged }: { courier: User; onchanged: (user: User) => void } = $props();

  let active = $state(untrack(() => courier.is_active));
  let toggling = $state(false);
  let toggleMessage = $state('');

  let resetOpen = $state(false);
  let newPassword = $state('');
  let resetting = $state(false);
  let resetError = $state('');
  let resetMessage = $state('');

  async function toggleActive(checked: boolean) {
    toggling = true;
    toggleMessage = '';
    try {
      onchanged(
        await apiFetch<User>(`/users/${courier.id}`, {
          method: 'PATCH',
          body: { is_active: checked }
        })
      );
      pushToast('success', checked ? 'Kurir diaktifkan.' : 'Kurir dinonaktifkan.');
    } catch (error) {
      active = courier.is_active;
      toggleMessage = errorMessage(error);
    } finally {
      toggling = false;
    }
  }

  async function resetPassword(event: SubmitEvent) {
    event.preventDefault();
    resetting = true;
    resetError = '';
    resetMessage = '';
    try {
      await apiFetch(`/users/${courier.id}/reset-password`, { body: { password: newPassword } });
      newPassword = '';
      resetOpen = false;
      pushToast('success', 'Kata sandi kurir disetel ulang.');
    } catch (error) {
      resetError = fieldErrors(error).password ?? '';
      if (!resetError) resetMessage = errorMessage(error);
    } finally {
      resetting = false;
    }
  }
</script>

<li class="flex flex-col gap-3 py-4">
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0">
      <p class="truncate font-semibold">{courier.name}</p>
      <p class="text-muted">{courier.phone}</p>
      <span
        class="mt-1 inline-block rounded-full px-2 py-0.5 text-sm font-medium {courier.is_active
          ? 'bg-status-done-soft text-status-done'
          : 'bg-neutral-200 text-neutral-700 dark:bg-neutral-700 dark:text-neutral-200'}"
      >
        {courier.is_active ? 'Aktif' : 'Nonaktif'}
      </span>
    </div>
    <Toggle label="Aktif" bind:checked={active} disabled={toggling} onchange={toggleActive} />
  </div>
  {#if toggleMessage}
    <Alert kind="error">{toggleMessage}</Alert>
  {/if}

  {#if resetOpen}
    <form class="flex flex-col gap-3" onsubmit={resetPassword} novalidate>
      <PasswordField
        label="Kata sandi baru untuk {courier.name}"
        bind:value={newPassword}
        error={resetError}
        hint="Minimal 10 karakter."
        autocomplete="new-password"
        minlength={10}
      />
      {#if resetMessage}
        <Alert kind="error">{resetMessage}</Alert>
      {/if}
      <div class="flex gap-2">
        <Button type="submit" loading={resetting}>Simpan kata sandi</Button>
        <Button variant="ghost" disabled={resetting} onclick={() => (resetOpen = false)}>
          Batal
        </Button>
      </div>
    </form>
  {:else}
    <div>
      <Button variant="secondary" onclick={() => (resetOpen = true)}>Setel ulang kata sandi</Button>
    </div>
  {/if}
</li>
