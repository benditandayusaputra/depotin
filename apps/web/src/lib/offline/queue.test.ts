import { describe, expect, it, vi } from 'vitest';
import { memoryStore, OfflineQueue, type QueuedAction } from './queue';

function action(id: string, createdAt: number): QueuedAction {
  return { id, orderId: `order-${id}`, action: 'deliver', body: {}, idempotencyKey: id, createdAt };
}

describe('OfflineQueue', () => {
  it('sends in creation order and drops actions answered with 200 or 409', async () => {
    const store = memoryStore();
    await store.put(action('b', 2));
    await store.put(action('a', 1));
    const send = vi.fn(async (item: QueuedAction) => (item.id === 'a' ? 409 : 200));
    const queue = new OfflineQueue(store, send);

    expect(await queue.flush()).toBe(0);
    expect(send.mock.calls.map(([item]) => item.id)).toEqual(['a', 'b']);
    expect(await store.all()).toEqual([]);
  });

  it('stops at a network error and keeps the rest for later', async () => {
    const store = memoryStore();
    await store.put(action('a', 1));
    await store.put(action('b', 2));
    const send = vi.fn(async () => {
      throw new TypeError('offline');
    });
    const queue = new OfflineQueue(store, send);

    expect(await queue.flush()).toBe(2);
    expect(send).toHaveBeenCalledTimes(1);
  });

  it('keeps an action the server failed to process and sends it later', async () => {
    const store = memoryStore();
    const queue = new OfflineQueue(store, async () => 503);
    await queue.enqueue({ orderId: 'o', action: 'dispatch', body: undefined, idempotencyKey: 'k' });

    expect(await queue.flush()).toBe(1);
    const [kept] = await store.all();
    expect(kept?.idempotencyKey).toBe('k');
  });

  it('runs only one flush at a time', async () => {
    const store = memoryStore();
    await store.put(action('a', 1));
    const send = vi.fn(async () => 200);
    const queue = new OfflineQueue(store, send);

    await Promise.all([queue.flush(), queue.flush()]);
    expect(send).toHaveBeenCalledTimes(1);
  });
});
