import { ApiError, apiFetch } from '$lib/api/client';
import type { components } from '$lib/api/schema';

export type User = components['schemas']['User'];
export type Depot = components['schemas']['Depot'];
export type DepotUpdate = components['schemas']['DepotUpdate'];
export type SessionData = { user: User; depot: Depot };
export type SessionStatus = 'unknown' | 'loading' | 'authenticated' | 'anonymous';

let user = $state<User | null>(null);
let depot = $state<Depot | null>(null);
let status = $state<SessionStatus>('unknown');

function setSession(data: SessionData): void {
  user = data.user;
  depot = data.depot;
  status = 'authenticated';
}

function clearSession(): void {
  user = null;
  depot = null;
  status = 'anonymous';
}

async function saveDepot(update: DepotUpdate): Promise<Depot> {
  depot = await apiFetch<Depot>('/depot', { method: 'PATCH', body: update });
  return depot;
}

async function loadSession(): Promise<void> {
  status = 'loading';
  try {
    setSession(await apiFetch<SessionData>('/auth/me'));
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) clearSession();
    else status = 'unknown';
    throw error;
  }
}

async function logout(): Promise<void> {
  try {
    await apiFetch('/auth/logout', { method: 'POST' });
  } finally {
    clearSession();
  }
}

export const session = {
  get user() {
    return user;
  },
  get depot() {
    return depot;
  },
  get status() {
    return status;
  },
  setSession,
  clearSession,
  saveDepot,
  loadSession,
  logout
};
