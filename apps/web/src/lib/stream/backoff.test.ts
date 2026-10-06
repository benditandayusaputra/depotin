import { describe, expect, it } from 'vitest';
import { reconnectDelay, shouldPoll } from './backoff';

describe('reconnectDelay', () => {
  it('follows the 1s, 2s schedule before switching to polling', () => {
    expect(reconnectDelay(1)).toBe(1000);
    expect(reconnectDelay(2)).toBe(2000);
    expect(shouldPoll(2)).toBe(false);
  });

  it('retries SSE every 60s once polling after three failures', () => {
    expect(shouldPoll(3)).toBe(true);
    expect(reconnectDelay(3)).toBe(60000);
    expect(reconnectDelay(10)).toBe(60000);
  });
});
