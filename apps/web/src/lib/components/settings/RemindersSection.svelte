<script lang="ts">
  import { untrack } from 'svelte';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import { session, type Depot } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';

  let { depot }: { depot: Depot } = $props();

  let leadDays = $state(untrack(() => String(depot.reminder_lead_days)));
  let daysPerGallon = $state(untrack(() => String(depot.default_days_per_gallon)));
  let saving = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');

  async function save(event: SubmitEvent) {
    event.preventDefault();
    saving = true;
    errors = {};
    message = '';
    try {
      await session.saveDepot({
        reminder_lead_days: Number(leadDays),
        default_days_per_gallon: Number(daysPerGallon)
      });
      pushToast('success', 'Pengaturan pengingat tersimpan.');
    } catch (error) {
      errors = fieldErrors(error);
      if (!hasFieldErrors(errors)) message = errorMessage(error);
    } finally {
      saving = false;
    }
  }
</script>

<Card>
  <form class="flex flex-col gap-4" onsubmit={save} novalidate>
    <TextField
      label="Kirim pengingat berapa hari sebelum galon habis?"
      bind:value={leadDays}
      error={errors.reminder_lead_days}
      hint="0 sampai 7 hari."
      type="number"
      inputmode="numeric"
      min="0"
      max="7"
      step="1"
      required
    />
    <TextField
      label="Perkiraan hari per galon untuk pelanggan baru"
      bind:value={daysPerGallon}
      error={errors.default_days_per_gallon}
      hint="Dipakai sampai pola pelanggan terbaca."
      type="number"
      inputmode="decimal"
      min="0.5"
      step="0.5"
      required
    />

    {#if message}
      <Alert kind="error">{message}</Alert>
    {/if}

    <Button type="submit" size="lg" loading={saving}>Simpan pengingat</Button>
  </form>
</Card>
