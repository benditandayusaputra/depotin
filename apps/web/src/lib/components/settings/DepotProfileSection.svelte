<script lang="ts">
  import { untrack } from 'svelte';
  import Alert from '$lib/components/Alert.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import Toggle from '$lib/components/Toggle.svelte';
  import { session, type Depot } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';

  let { depot }: { depot: Depot } = $props();

  const initial = untrack(() => ({
    name: depot.name,
    phone: depot.phone,
    address: depot.address,
    openTime: depot.open_time,
    closeTime: depot.close_time,
    deliveryFee: String(depot.delivery_fee),
    acceptingOrders: depot.is_accepting_orders,
    autoConfirmKnown: depot.auto_confirm_known
  }));

  let name = $state(initial.name);
  let phone = $state(initial.phone);
  let address = $state(initial.address);
  let openTime = $state(initial.openTime);
  let closeTime = $state(initial.closeTime);
  let deliveryFee = $state(initial.deliveryFee);
  let acceptingOrders = $state(initial.acceptingOrders);
  let autoConfirmKnown = $state(initial.autoConfirmKnown);
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
        name,
        phone,
        address,
        open_time: openTime,
        close_time: closeTime,
        delivery_fee: Number(deliveryFee),
        is_accepting_orders: acceptingOrders,
        auto_confirm_known: autoConfirmKnown
      });
      pushToast('success', 'Profil depot tersimpan.');
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
    <TextField label="Nama depot" bind:value={name} error={errors.name} required />
    <TextField
      label="Nomor HP depot"
      bind:value={phone}
      error={errors.phone}
      type="tel"
      inputmode="tel"
      hint="Dipakai pelanggan untuk menghubungi lewat WhatsApp."
      required
    />
    <TextField label="Alamat" bind:value={address} error={errors.address} />
    <div class="grid grid-cols-2 gap-4">
      <TextField
        label="Jam buka"
        bind:value={openTime}
        error={errors.open_time}
        type="time"
        required
      />
      <TextField
        label="Jam tutup"
        bind:value={closeTime}
        error={errors.close_time}
        type="time"
        required
      />
    </div>
    <TextField
      label="Ongkos kirim (Rp)"
      bind:value={deliveryFee}
      error={errors.delivery_fee}
      type="number"
      inputmode="numeric"
      min="0"
      step="500"
      required
    />
    <Toggle label="Menerima pesanan" bind:checked={acceptingOrders} />
    <Toggle
      label="Pesanan dari link pribadi langsung dikonfirmasi"
      bind:checked={autoConfirmKnown}
    />

    {#if message}
      <Alert kind="error">{message}</Alert>
    {/if}

    <Button type="submit" size="lg" loading={saving}>Simpan profil</Button>
  </form>
</Card>
