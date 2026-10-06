<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { apiFetch } from '$lib/api/client';
  import { apiFetchPage, withCursor } from '$lib/api/page';
  import type { Customer, CustomerUpdate, LedgerEntry, LinkInfo, Order } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import ConfidenceBadge from '$lib/components/ConfidenceBadge.svelte';
  import CustomerForm from '$lib/components/CustomerForm.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import LinkButton from '$lib/components/LinkButton.svelte';
  import OrderCard from '$lib/components/OrderCard.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SelectField from '$lib/components/SelectField.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import SnoozeMenu from '$lib/components/SnoozeMenu.svelte';
  import TextField from '$lib/components/TextField.svelte';
  import Toggle from '$lib/components/Toggle.svelte';
  import { session } from '$lib/state/session.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { formatDate, formatDateTime, relativeDays } from '$lib/utils/date';
  import { errorMessage, fieldErrors, hasFieldErrors } from '$lib/utils/errors';
  import { predictionSentence } from '$lib/utils/prediction';

  type LedgerKind = 'adjustment' | 'lost';

  const LEDGER_KIND_LABELS: Record<LedgerEntry['kind'], string> = {
    delivery: 'Pengantaran',
    adjustment: 'Penyesuaian',
    lost: 'Hilang'
  };

  const customerId = $derived(page.params.id ?? '');

  let customer = $state<Customer | null>(null);
  let loading = $state(true);
  let loadError = $state('');

  let ledger = $state<LedgerEntry[]>([]);
  let ledgerCursor = $state<string | null>(null);
  let ledgerLoading = $state(true);
  let ledgerError = $state('');

  let orders = $state<Order[]>([]);
  let ordersCursor = $state<string | null>(null);
  let ordersLoading = $state(true);
  let ordersError = $state('');

  let link = $state<LinkInfo | null>(null);
  let linkBusy = $state('');
  let linkError = $state('');
  let rotateConfirm = $state(false);
  let linkInput = $state<HTMLInputElement | null>(null);

  let snoozing = $state(false);

  let adjustKind = $state<LedgerKind>('adjustment');
  let adjustDelta = $state('');
  let adjustNote = $state('');
  let adjusting = $state(false);
  let adjustErrors = $state<Record<string, string>>({});
  let adjustMessage = $state('');

  let editing = $state(false);
  let saving = $state(false);
  let editErrors = $state<Record<string, string>>({});
  let editMessage = $state('');
  let active = $state(true);
  let toggling = $state(false);
  let toggleMessage = $state('');

  const stampsEvery = $derived(session.depot?.loyalty_every ?? null);

  async function load() {
    loading = true;
    loadError = '';
    try {
      customer = await apiFetch<Customer>(`/customers/${customerId}`);
      active = customer.is_active;
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  async function loadLedger(more = false) {
    ledgerLoading = true;
    ledgerError = '';
    try {
      const base = `/customers/${customerId}/ledger`;
      const result = await apiFetchPage<LedgerEntry>(more ? withCursor(base, ledgerCursor) : base);
      ledger = more ? [...ledger, ...result.data] : result.data;
      ledgerCursor = result.nextCursor;
    } catch (error) {
      ledgerError = errorMessage(error);
    } finally {
      ledgerLoading = false;
    }
  }

  async function loadOrders(more = false) {
    ordersLoading = true;
    ordersError = '';
    try {
      const base = `/customers/${customerId}/orders`;
      const result = await apiFetchPage<Order>(more ? withCursor(base, ordersCursor) : base);
      orders = more ? [...orders, ...result.data] : result.data;
      ordersCursor = result.nextCursor;
    } catch (error) {
      ordersError = errorMessage(error);
    } finally {
      ordersLoading = false;
    }
  }

  async function ensureLink(): Promise<LinkInfo | null> {
    if (link) return link;
    try {
      link = await apiFetch<LinkInfo>(`/customers/${customerId}/link`, { method: 'POST' });
      if (customer) customer = { ...customer, has_link: true };
      return link;
    } catch (error) {
      linkError = errorMessage(error);
      return null;
    }
  }

  async function copyLink() {
    linkBusy = 'copy';
    linkError = '';
    const info = await ensureLink();
    linkBusy = '';
    if (!info) return;
    try {
      await navigator.clipboard.writeText(info.link);
      pushToast('success', 'Link disalin.');
    } catch {
      linkInput?.select();
      pushToast('info', 'Pilih teks link lalu salin manual.');
    }
  }

  async function shareLink() {
    linkBusy = 'share';
    linkError = '';
    const info = await ensureLink();
    linkBusy = '';
    if (info) window.open(info.wa_url, '_blank', 'noopener');
  }

  async function rotateLink() {
    linkBusy = 'rotate';
    linkError = '';
    try {
      link = await apiFetch<LinkInfo>(`/customers/${customerId}/link/rotate`, { method: 'POST' });
      if (customer) customer = { ...customer, has_link: true };
      rotateConfirm = false;
      pushToast('success', 'Link baru dibuat. Link lama tidak berlaku.');
    } catch (error) {
      linkError = errorMessage(error);
    } finally {
      linkBusy = '';
    }
  }

  async function snooze(days: number) {
    snoozing = true;
    try {
      const result = await apiFetch<{ snoozed_until: string }>(`/customers/${customerId}/snooze`, {
        body: { days }
      });
      if (customer) customer = { ...customer, reminder_snoozed_until: result.snoozed_until };
      pushToast('success', `Pengingat ditunda ${days} hari.`);
    } catch (error) {
      pushToast('error', errorMessage(error));
    } finally {
      snoozing = false;
    }
  }

  async function adjust(event: SubmitEvent) {
    event.preventDefault();
    adjusting = true;
    adjustErrors = {};
    adjustMessage = '';
    try {
      const entry = await apiFetch<LedgerEntry>(`/customers/${customerId}/ledger-adjustments`, {
        body: { kind: adjustKind, delta: Number(adjustDelta), note: adjustNote.trim() }
      });
      ledger = [entry, ...ledger];
      if (customer) customer = { ...customer, loan_balance: entry.balance_after };
      adjustDelta = '';
      adjustNote = '';
      pushToast('success', 'Buku galon diperbarui.');
    } catch (error) {
      adjustErrors = fieldErrors(error);
      if (!hasFieldErrors(adjustErrors)) adjustMessage = errorMessage(error);
    } finally {
      adjusting = false;
    }
  }

  async function saveEdit(values: CustomerUpdate) {
    saving = true;
    editErrors = {};
    editMessage = '';
    try {
      customer = await apiFetch<Customer>(`/customers/${customerId}`, {
        method: 'PATCH',
        body: values
      });
      editing = false;
      pushToast('success', 'Data pelanggan disimpan.');
    } catch (error) {
      editErrors = fieldErrors(error);
      if (!hasFieldErrors(editErrors)) editMessage = errorMessage(error);
    } finally {
      saving = false;
    }
  }

  async function toggleActive(checked: boolean) {
    toggling = true;
    toggleMessage = '';
    try {
      customer = await apiFetch<Customer>(`/customers/${customerId}`, {
        method: 'PATCH',
        body: { is_active: checked }
      });
      pushToast('success', checked ? 'Pelanggan diaktifkan.' : 'Pelanggan dinonaktifkan.');
    } catch (error) {
      active = customer?.is_active ?? true;
      toggleMessage = errorMessage(error);
    } finally {
      toggling = false;
    }
  }

  onMount(() => {
    void load();
    void loadLedger();
    void loadOrders();
  });
</script>

<svelte:head>
  <title>{customer ? customer.name : 'Pelanggan'} | Depotin</title>
</svelte:head>

<a
  href={resolve('/app/pelanggan')}
  class="mb-2 inline-flex min-h-11 items-center font-semibold text-accent-700 dark:text-accent-300"
>
  &larr; Semua pelanggan
</a>

{#if loading}
  <Skeleton lines={10} />
{:else if loadError || !customer}
  <Alert kind="error" onretry={() => void load()}>{loadError}</Alert>
{:else}
  <PageHeader title={customer.name} description={customer.area || undefined}>
    {#snippet actions()}
      {#if customer}
        <Badge tone={customer.is_active ? 'done' : 'neutral'}>
          {customer.is_active ? 'Aktif' : 'Nonaktif'}
        </Badge>
      {/if}
    {/snippet}
  </PageHeader>

  <div class="flex flex-col gap-4">
    <Card>
      <p class="text-lg font-semibold">{customer.phone}</p>
      <p class="mt-1">{customer.address || 'Alamat belum diisi'}</p>
      {#if customer.address_note}
        <p class="text-muted">{customer.address_note}</p>
      {/if}
      <div class="mt-3 flex flex-wrap gap-2">
        <LinkButton href="tel:+{customer.phone}" variant="secondary">Telepon</LinkButton>
        <LinkButton href="https://wa.me/{customer.phone}" variant="secondary" external>
          WhatsApp
        </LinkButton>
      </div>
    </Card>

    <Card>
      <h2 class="mb-2 text-lg font-semibold">Perkiraan galon habis</h2>
      <p class="text-lg">{predictionSentence(customer)}</p>
      {#if customer.predicted_empty_at}
        <p class="mt-1 text-muted">
          Diperkirakan habis {relativeDays(customer.predicted_empty_at)} ({formatDate(
            customer.predicted_empty_at
          )})
        </p>
      {/if}
      <div class="mt-3 flex flex-wrap items-center gap-2">
        <ConfidenceBadge confidence={customer.prediction_confidence} />
        <Badge>Biasa pesan {customer.usual_qty} galon</Badge>
      </div>
    </Card>

    <div class="grid grid-cols-2 gap-3">
      <Card>
        <p class="text-sm text-muted">Galon dipinjam</p>
        <p class="text-num">{customer.loan_balance}</p>
      </Card>
      <Card>
        <p class="text-sm text-muted">Stempel loyalitas</p>
        <p class="text-num">
          {customer.stamp_count}{#if stampsEvery}<span class="text-lg font-medium text-muted"
              >/{stampsEvery}</span
            >{/if}
        </p>
      </Card>
    </div>

    <Card>
      <h2 class="mb-2 text-lg font-semibold">Link pribadi</h2>
      <p class="mb-3 text-muted">
        {customer.has_link
          ? 'Pelanggan sudah punya link untuk pesan ulang sekali ketuk.'
          : 'Link akan dibuat saat pertama kali disalin atau dikirim.'}
      </p>
      {#if link}
        <input
          bind:this={linkInput}
          readonly
          value={link.link}
          aria-label="Link pribadi"
          class="mb-3 min-h-12 w-full rounded-lg border border-border bg-neutral-50 px-3 text-sm text-text dark:bg-neutral-800"
          onfocus={(event) => event.currentTarget.select()}
        />
      {/if}
      {#if linkError}
        <div class="mb-3"><Alert kind="error">{linkError}</Alert></div>
      {/if}
      <div class="flex flex-wrap gap-2">
        <Button variant="secondary" loading={linkBusy === 'copy'} onclick={() => void copyLink()}>
          Salin link
        </Button>
        <Button loading={linkBusy === 'share'} onclick={() => void shareLink()}>
          Kirim lewat WhatsApp
        </Button>
        <Button variant="ghost" onclick={() => (rotateConfirm = !rotateConfirm)}
          >Buat link baru</Button
        >
      </div>
      {#if rotateConfirm}
        <div class="mt-3 flex flex-col gap-3">
          <Alert kind="error"
            >Link lama langsung tidak berlaku. Pelanggan perlu link yang baru.</Alert
          >
          <div class="flex gap-2">
            <Button
              variant="danger"
              loading={linkBusy === 'rotate'}
              onclick={() => void rotateLink()}
            >
              Ya, buat link baru
            </Button>
            <Button variant="ghost" onclick={() => (rotateConfirm = false)}>Jangan</Button>
          </div>
        </div>
      {/if}
    </Card>

    <Card>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-lg font-semibold">Pengingat</h2>
          <p class="text-muted">
            {customer.reminder_snoozed_until
              ? `Ditunda sampai ${formatDate(`${customer.reminder_snoozed_until}T12:00:00+07:00`)}`
              : 'Pengingat berjalan normal.'}
          </p>
        </div>
        <SnoozeMenu busy={snoozing} onpick={(days) => void snooze(days)} />
      </div>
    </Card>

    <Card>
      <h2 class="mb-3 text-lg font-semibold">Buku galon</h2>
      {#if ledgerLoading && ledger.length === 0}
        <Skeleton lines={3} />
      {:else if ledgerError}
        <Alert kind="error" onretry={() => void loadLedger()}>{ledgerError}</Alert>
      {:else if ledger.length === 0}
        <p class="text-muted">Belum ada catatan.</p>
      {:else}
        <ul class="divide-y divide-border">
          {#each ledger as entry (entry.id)}
            <li class="flex items-center justify-between gap-3 py-2">
              <div class="min-w-0">
                <p class="font-medium">{LEDGER_KIND_LABELS[entry.kind]}</p>
                <p class="text-sm text-muted">
                  {formatDateTime(entry.created_at)}{entry.note ? ` · ${entry.note}` : ''}
                </p>
              </div>
              <div class="text-right">
                <p class="font-bold {entry.delta > 0 ? 'text-status-pending' : 'text-status-done'}">
                  {entry.delta > 0 ? '+' : ''}{entry.delta}
                </p>
                <p class="text-sm text-muted">saldo {entry.balance_after}</p>
              </div>
            </li>
          {/each}
        </ul>
        {#if ledgerCursor}
          <div class="mt-3">
            <Button
              variant="secondary"
              loading={ledgerLoading}
              onclick={() => void loadLedger(true)}
            >
              Muat lebih banyak
            </Button>
          </div>
        {/if}
      {/if}

      <form
        class="mt-4 flex flex-col gap-3 border-t border-border pt-4"
        onsubmit={adjust}
        novalidate
      >
        <h3 class="font-semibold">Penyesuaian manual</h3>
        <SelectField label="Jenis" bind:value={adjustKind} error={adjustErrors.kind}>
          <option value="adjustment">Penyesuaian</option>
          <option value="lost">Galon hilang</option>
        </SelectField>
        <TextField
          label="Perubahan galon"
          bind:value={adjustDelta}
          type="number"
          inputmode="numeric"
          placeholder="Misal: -1 bila dikembalikan"
          hint="Angka minus berarti galon kembali ke depot."
          error={adjustErrors.delta}
          required
        />
        <TextField
          label="Catatan"
          bind:value={adjustNote}
          error={adjustErrors.note}
          placeholder="Wajib diisi"
          required
        />
        {#if adjustMessage}
          <Alert kind="error">{adjustMessage}</Alert>
        {/if}
        <Button
          type="submit"
          variant="secondary"
          loading={adjusting}
          disabled={!adjustDelta || adjustNote.trim().length < 3}
        >
          Simpan penyesuaian
        </Button>
      </form>
    </Card>

    <Card>
      <h2 class="mb-3 text-lg font-semibold">Riwayat pesanan</h2>
      {#if ordersLoading && orders.length === 0}
        <Skeleton lines={4} />
      {:else if ordersError}
        <Alert kind="error" onretry={() => void loadOrders()}>{ordersError}</Alert>
      {:else if orders.length === 0}
        <EmptyState title="Belum ada pesanan." />
      {:else}
        <ul class="flex flex-col gap-3">
          {#each orders as order (order.id)}
            <OrderCard {order} />
          {/each}
        </ul>
        {#if ordersCursor}
          <div class="mt-3">
            <Button
              variant="secondary"
              loading={ordersLoading}
              onclick={() => void loadOrders(true)}
            >
              Muat lebih banyak
            </Button>
          </div>
        {/if}
      {/if}
    </Card>

    <Card>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-lg font-semibold">Data pelanggan</h2>
        <Toggle label="Aktif" bind:checked={active} disabled={toggling} onchange={toggleActive} />
      </div>
      {#if toggleMessage}
        <div class="mt-2"><Alert kind="error">{toggleMessage}</Alert></div>
      {/if}
      <div class="mt-3">
        {#if editing}
          <CustomerForm
            initial={customer}
            submitLabel="Simpan perubahan"
            {saving}
            errors={editErrors}
            message={editMessage}
            onsubmit={(values) => void saveEdit(values)}
            oncancel={() => (editing = false)}
          />
        {:else}
          <Button variant="secondary" onclick={() => (editing = true)}>Ubah data</Button>
        {/if}
      </div>
    </Card>
  </div>
{/if}
