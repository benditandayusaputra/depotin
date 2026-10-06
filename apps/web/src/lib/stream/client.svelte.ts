import { SvelteSet } from 'svelte/reactivity';
import { env } from '$env/dynamic/public';
import { apiFetch } from '$lib/api/client';
import type { components } from '$lib/api/schema';
import { POLL_INTERVAL_MS, reconnectDelay, shouldPoll } from './backoff';

export type StreamStatus = 'connecting' | 'connected' | 'reconnecting' | 'polling' | 'off';

export interface OrderEvent {
  id: string;
  code: string;
  status: components['schemas']['OrderStatus'];
  source: components['schemas']['OrderSource'];
  delivery_name: string;
  total: number;
  courier_id: string | null;
}

export type StreamEvent =
  | { type: 'order.created'; data: OrderEvent }
  | { type: 'order.updated'; data: OrderEvent }
  | { type: 'reminder.queued'; data: { count: number } }
  | { type: 'poll' };

export type StreamHandler = (event: StreamEvent) => void;

const STREAM_ORIGIN = env.PUBLIC_STREAM_ORIGIN || 'http://localhost:8080';
const DATA_EVENTS = ['order.created', 'order.updated', 'reminder.queued'] as const;

let status = $state<StreamStatus>('off');
let source: EventSource | null = null;
let failures = 0;
let generation = 0;
let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
let pollTimer: ReturnType<typeof setInterval> | undefined;
const handlers = new SvelteSet<StreamHandler>();

function emit(event: StreamEvent): void {
  for (const handler of handlers) handler(event);
}

function parseEvent(type: (typeof DATA_EVENTS)[number], raw: string): StreamEvent | null {
  try {
    const data: unknown = JSON.parse(raw);
    if (typeof data !== 'object' || data === null) return null;
    if (type === 'reminder.queued') return { type, data: data as { count: number } };
    return { type, data: data as OrderEvent };
  } catch {
    return null;
  }
}

function stopPolling(): void {
  clearInterval(pollTimer);
  pollTimer = undefined;
}

function startPolling(): void {
  if (pollTimer) return;
  pollTimer = setInterval(() => emit({ type: 'poll' }), POLL_INTERVAL_MS);
}

function closeSource(): void {
  source?.close();
  source = null;
}

function scheduleReconnect(currentGeneration: number): void {
  if (currentGeneration !== generation) return;
  closeSource();
  failures += 1;
  if (shouldPoll(failures)) {
    status = 'polling';
    startPolling();
  } else {
    status = 'reconnecting';
  }
  clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(() => void open(currentGeneration), reconnectDelay(failures));
}

async function open(currentGeneration: number): Promise<void> {
  if (currentGeneration !== generation) return;
  let ticket: string;
  try {
    const response = await apiFetch<{ ticket: string; expires_in: number }>('/stream/tickets', {
      method: 'POST'
    });
    ticket = response.ticket;
  } catch {
    scheduleReconnect(currentGeneration);
    return;
  }
  if (currentGeneration !== generation) return;

  const eventSource = new EventSource(
    `${STREAM_ORIGIN}/api/v1/stream?ticket=${encodeURIComponent(ticket)}`
  );
  source = eventSource;
  eventSource.addEventListener('ready', () => {
    if (currentGeneration !== generation) return;
    failures = 0;
    stopPolling();
    status = 'connected';
  });
  for (const type of DATA_EVENTS) {
    eventSource.addEventListener(type, (message: MessageEvent<string>) => {
      const event = parseEvent(type, message.data);
      if (event && currentGeneration === generation) emit(event);
    });
  }
  eventSource.onerror = () => {
    if (source === eventSource) scheduleReconnect(currentGeneration);
  };
}

function connect(): void {
  if (status !== 'off') return;
  generation += 1;
  failures = 0;
  status = 'connecting';
  void open(generation);
}

function disconnect(): void {
  generation += 1;
  clearTimeout(reconnectTimer);
  stopPolling();
  closeSource();
  failures = 0;
  status = 'off';
}

function subscribe(handler: StreamHandler): () => void {
  handlers.add(handler);
  return () => {
    handlers.delete(handler);
  };
}

export const stream = {
  get status() {
    return status;
  },
  connect,
  disconnect,
  subscribe
};
