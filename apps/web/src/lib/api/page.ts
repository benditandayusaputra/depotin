import { apiFetchPayload } from './client';

export interface Page<T> {
  data: T[];
  nextCursor: string | null;
}

function readCursor(payload: unknown): string | null {
  if (typeof payload !== 'object' || payload === null || !('meta' in payload)) return null;
  const meta = payload.meta;
  if (typeof meta !== 'object' || meta === null || !('next_cursor' in meta)) return null;
  return typeof meta.next_cursor === 'string' ? meta.next_cursor : null;
}

export function withCursor(path: string, cursor: string | null): string {
  if (!cursor) return path;
  return `${path}${path.includes('?') ? '&' : '?'}cursor=${encodeURIComponent(cursor)}`;
}

export async function apiFetchPage<T>(path: string): Promise<Page<T>> {
  const payload = await apiFetchPayload(path);
  const data =
    typeof payload === 'object' && payload !== null && 'data' in payload
      ? (payload.data as T[])
      : [];
  return { data, nextCursor: readCursor(payload) };
}
