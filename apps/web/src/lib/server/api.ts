import { env } from '$env/dynamic/private';
import { error } from '@sveltejs/kit';

const UNAVAILABLE_MESSAGE = 'Server sedang tidak bisa dihubungi. Coba lagi sebentar.';
const TIMEOUT_MS = 10_000;

interface ClientEvent {
  getClientAddress(): string;
}

async function upstreamMessage(response: Response): Promise<string> {
  const parsed: unknown = await response.json().catch(() => null);
  if (typeof parsed !== 'object' || parsed === null || !('error' in parsed)) {
    return UNAVAILABLE_MESSAGE;
  }
  const detail = parsed.error;
  if (typeof detail === 'object' && detail !== null && 'message' in detail) {
    if (typeof detail.message === 'string') return detail.message;
  }
  return UNAVAILABLE_MESSAGE;
}

export async function serverApi<T>(event: ClientEvent, path: string): Promise<T> {
  const headers = new Headers({
    accept: 'application/json',
    'x-client-ip': event.getClientAddress(),
    'x-request-id': crypto.randomUUID()
  });
  if (env.EDGE_KEY) headers.set('x-edge-key', env.EDGE_KEY);
  const origin = env.API_ORIGIN ?? 'http://localhost:8080';

  let response: Response;
  try {
    response = await fetch(`${origin}/api/v1${path}`, {
      headers,
      signal: AbortSignal.timeout(TIMEOUT_MS)
    });
  } catch {
    error(502, UNAVAILABLE_MESSAGE);
  }

  if (response.status === 404) error(404, 'Link tidak dikenal.');
  if (!response.ok) error(response.status === 429 ? 429 : 502, await upstreamMessage(response));
  const payload = (await response.json()) as { data: T };
  return payload.data;
}
