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

  let loyaltyEvery = $state(untrack(() => String(depot.loyalty_every ?? 0)));
  let saving = $state(false);
  let errors = $state<Record<string, string>>({});
  let message = $state('');

  const count = $derived(Number(loyaltyEvery));
  const summary = $derived(
    count > 0 ? `Isi ${count} kali gratis 1.` : 'Program loyalitas dimatikan.'
  );

  async function save(event: SubmitEvent) {
    event.preventDefault();
    saving = true;
    errors = {};
    message = '';
    try {
      await session.saveDepot({ loyalty_every: count });
      pushToast('success', 'Pengaturan loyalitas tersimpan.');
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
      label="Gratis 1 galon setiap berapa kali isi?"
      bind:value={loyaltyEvery}
      error={errors.loyalty_every}
      hint="Isi 0 untuk mematikan program loyalitas."
      type="number"
      inputmode="numeric"
      min="0"
      step="1"
      required
    />
    <p class="text-lg font-semibold">{summary}</p>

    {#if message}
      <Alert kind="error">{message}</Alert>
    {/if}

    <Button type="submit" size="lg" loading={saving}>Simpan loyalitas</Button>
  </form>
</Card>
