<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import CourierRow from '$lib/components/settings/CourierRow.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import PasswordField from '$lib/components/PasswordField.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import type { User } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';

  let couriers = $state<User[]>([]);
  let loading = $state(true);
  let loadError = $state('');

  let name = $state('');
  let phone = $state('');
  let password = $state('');
  let adding = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');

  async function load() {
    loading = true;
    loadError = '';
    try {
      const users = await apiFetch<User[]>('/users');
      couriers = users.filter((user) => user.role === 'courier');
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  function replaceCourier(updated: User) {
    couriers = couriers.map((courier) => (courier.id === updated.id ? updated : courier));
  }

  async function add(event: SubmitEvent) {
    event.preventDefault();
    adding = true;
    errors = {};
    message = '';
    try {
      const created = await apiFetch<User>('/users', { body: { name, phone, password } });
      couriers = [...couriers, created];
      name = '';
      phone = '';
      password = '';
      pushToast('success', 'Kurir ditambahkan.');
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      adding = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<div class="flex flex-col gap-4">
  <Card>
    <h2 class="mb-3 text-lg font-semibold">Daftar kurir</h2>
    {#if loading}
      <Skeleton lines={4} />
    {:else if loadError}
      <Alert kind="error" onretry={load}>{loadError}</Alert>
    {:else if couriers.length === 0}
      <EmptyState
        title="Belum ada kurir"
        description="Tambahkan kurir supaya pesanan bisa ditugaskan."
      />
    {:else}
      <ul class="divide-y divide-border">
        {#each couriers as courier (courier.id)}
          <CourierRow {courier} onchanged={replaceCourier} />
        {/each}
      </ul>
    {/if}
  </Card>

  <Card>
    <h2 class="mb-3 text-lg font-semibold">Tambah kurir</h2>
    <form class="flex flex-col gap-4" onsubmit={add} novalidate>
      <TextField label="Nama kurir" bind:value={name} error={errors.name} required />
      <TextField
        label="Nomor HP kurir"
        bind:value={phone}
        error={errors.phone}
        type="tel"
        inputmode="tel"
        placeholder="08xxxxxxxxxx"
        required
      />
      <PasswordField
        label="Kata sandi kurir"
        bind:value={password}
        error={errors.password}
        hint="Minimal 10 karakter."
        autocomplete="new-password"
        minlength={10}
      />

      {#if message}
        <Alert kind="error">{message}</Alert>
      {/if}

      <Button type="submit" size="lg" loading={adding}>Tambah kurir</Button>
    </form>
  </Card>
</div>
