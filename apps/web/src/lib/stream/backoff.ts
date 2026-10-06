const RECONNECT_DELAYS_MS = [1000, 2000, 5000, 10000];
const MAX_RECONNECT_DELAY_MS = 30000;

export const FAILURES_BEFORE_POLLING = 3;
export const POLL_INTERVAL_MS = 10000;
export const SSE_RETRY_WHILE_POLLING_MS = 60000;

export function reconnectDelay(failures: number): number {
  if (failures >= FAILURES_BEFORE_POLLING) return SSE_RETRY_WHILE_POLLING_MS;
  return RECONNECT_DELAYS_MS[Math.max(failures - 1, 0)] ?? MAX_RECONNECT_DELAY_MS;
}

export function shouldPoll(failures: number): boolean {
  return failures >= FAILURES_BEFORE_POLLING;
}
