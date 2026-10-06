<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { apiFetch } from '$lib/api/client';
  import type { Reminder } from '$lib/api/types';
  import Alert from '$lib/components/Alert.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import ConfidenceBadge from '$lib/components/ConfidenceBadge.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import SnoozeMenu from '$lib/components/SnoozeMenu.svelte';
  import { pushToast } from '$lib/state/toast.svelte';
  import { stream } from '$lib/stream/client.svelte';
  import { relativeDays } from '$lib/utils/date';
  import { errorMessage } from '$lib/utils/errors';

  type SendResult = { reminder: Reminder; link: string; wa_url: string };

  let reminders = $state<Reminder[]>([]);
  let loading = $state(true);
  let loadError = $state('');
  let busy = $state<Record<string, string>>({});
  let rowErrors = $state<Record<string, string>>({});

  const sorted = $derived(
    [...reminders].sort((a, b) => Number(a.status === 'sent') - Number(b.status === 'sent'))
  );
  const queuedCount = $derived(reminders.filter((item) => item.status === 'queued').length);

  async function load(silent = false) {
    if (!silent) loading = true;
    loadError = '';
    try {
      reminders = await apiFetch<Reminder[]>('/reminders');
    } catch (error) {
      loadError = errorMessage(error);
    } finally {
      loading = false;
    }
  }

  function setBusy(id: string, action: string) {
    busy = { ...busy, [id]: action };
    rowErrors = { ...rowErrors, [id]: '' };
  }

  function clearBusy(id: string) {
    const next = { ...busy };
    delete next[id];
    busy = next;
  }

  async function send(reminder: Reminder) {
    setBusy(reminder.id, 'send');
    try {
      const result = await apiFetch<SendResult>(`/reminders/${reminder.id}/send`, {
        method: 'POST'
      });
      reminders = reminders.map((item) => (item.id === reminder.id ? result.reminder : item));
      window.open(result.wa_url, '_blank', 'noopener');
      pushToast('success', `Pengingat untuk ${reminder.customer.name} terkirim.`);
    } catch (error) {
      rowErrors = { ...rowErrors, [reminder.id]: errorMessage(error) };
    } finally {
      clearBusy(reminder.id);
    }
  }

  async function skip(reminder: Reminder) {
    setBusy(reminder.id, 'skip');
    try {
      await apiFetch(`/reminders/${reminder.id}/skip`, { method: 'POST' });
      reminders = reminders.filter((item) => item.id !== reminder.id);
      pushToast('success', `${reminder.customer.name} dilewati hari ini.`);
    } catch (error) {
      rowErrors = { ...rowErrors, [reminder.id]: errorMessage(error) };
    } finally {
      clearBusy(reminder.id);
    }
  }

  async function snooze(reminder: Reminder, days: number) {
    setBusy(reminder.id, 'snooze');
    try {
      await apiFetch(`/customers/${reminder.customer.id}/snooze`, { body: { days } });
      reminders = reminders.filter((item) => item.id !== reminder.id);
      pushToast('success', `${reminder.customer.name} ditunda ${days} hari.`);
    } catch (error) {
      rowErrors = { ...rowErrors, [reminder.id]: errorMessage(error) };
    } finally {
      clearBusy(reminder.id);
    }
  }

  onMount(() => {
    void load();
    return stream.subscribe((event) => {
      if (event.type === 'reminder.queued' || event.type === 'poll') void load(true);
    });
  });
</script>

<svelte:head>
  <title>Pengingat | Depotin</title>
</svelte:head>

<PageHeader
  title="Pengingat"
  description={loading ? undefined : `${queuedCount} pelanggan perlu diingatkan hari ini.`}
/>

{#if loading}
  <Skeleton lines={8} />
{:else if loadError}
  <Alert kind="error" onretry={() => void load()}>{loadError}</Alert>
{:else if sorted.length === 0}
  <EmptyState title="Tidak ada yang perlu diingatkan hari ini." />
{:else}
  <ul class="flex flex-col gap-3">
    {#each sorted as reminder (reminder.id)}
      {@const sent = reminder.status === 'sent'}
      <li class="flex flex-col gap-3 rounded-lg border border-border bg-surface p-4 shadow-card">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <a
              href={resolve('/app/pelanggan/[id]', { id: reminder.customer.id })}
              class="text-lg font-semibold text-accent-700 underline dark:text-accent-300"
            >
              {reminder.customer.name}
            </a>
            <p class="text-muted">
              {reminder.customer.area || 'Area belum diisi'} &middot; biasa {reminder.customer
                .usual_qty} galon
            </p>
            <p>Diperkirakan habis {relativeDays(reminder.predicted_empty_at)}</p>
          </div>
          {#if sent}
            <Badge tone="done">Terkirim</Badge>
          {:else}
            <Badge tone="pending">Menunggu</Badge>
          {/if}
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <ConfidenceBadge confidence={reminder.customer.prediction_confidence} />
        </div>
        {#if rowErrors[reminder.id]}
          <Alert kind="error">{rowErrors[reminder.id]}</Alert>
        {/if}
        {#if !sent}
          <div class="flex flex-col gap-2 sm:flex-row">
            <Button
              size="lg"
              class="sm:flex-1"
              loading={busy[reminder.id] === 'send'}
              disabled={Boolean(busy[reminder.id])}
              onclick={() => void send(reminder)}
            >
              Kirim WA
            </Button>
            <div class="flex gap-2">
              <Button
                variant="secondary"
                class="flex-1"
                loading={busy[reminder.id] === 'skip'}
                disabled={Boolean(busy[reminder.id])}
                onclick={() => void skip(reminder)}
              >
                Lewati
              </Button>
              <SnoozeMenu
                busy={busy[reminder.id] === 'snooze'}
                onpick={(days) => void snooze(reminder, days)}
              />
            </div>
          </div>
        {/if}
      </li>
    {/each}
  </ul>
{/if}
