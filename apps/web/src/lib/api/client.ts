import type { components } from './schema';

export type { paths } from './schema';

type ErrorBody = components['schemas']['ErrorBody'];

export type ApiErrorCode = ErrorBody['code'];

export class ApiError extends Error {
  readonly code: ApiErrorCode;
  readonly status: number;
  readonly fields: Record<string, string>;

  constructor(status: number, body: ErrorBody) {
    super(body.message);
    this.name = 'ApiError';
    this.code = body.code;
    this.status = status;
    this.fields = body.fields ?? {};
  }
}

export interface ApiFetchInit {
  method?: string;
  body?: unknown;
  idempotencyKey?: string;
  signal?: AbortSignal;
}

const API_BASE = '/api/v1';
const REFRESH_PATH = '/auth/refresh';
const NO_REFRESH_PATHS = new Set([REFRESH_PATH, '/auth/login', '/auth/register']);
const FALLBACK_ERROR: ErrorBody = { code: 'internal', message: 'Terjadi gangguan. Coba lagi.' };

let refreshInFlight: Promise<boolean> | null = null;

export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}

function isErrorResponse(value: unknown): value is { error: ErrorBody } {
  if (typeof value !== 'object' || value === null || !('error' in value)) return false;
  const error = value.error;
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    'message' in error &&
    typeof error.code === 'string' &&
    typeof error.message === 'string'
  );
}

function isEnvelope(value: unknown): value is { data: unknown } {
  return typeof value === 'object' && value !== null && 'data' in value;
}

async function readJson(response: Response): Promise<unknown> {
  return response.json().catch(() => null);
}

function refreshSession(): Promise<boolean> {
  refreshInFlight ??= fetch(`${API_BASE}${REFRESH_PATH}`, { method: 'POST' })
    .then((response) => response.ok)
    .catch(() => false)
    .finally(() => {
      refreshInFlight = null;
    });
  return refreshInFlight;
}

function sendRequest(path: string, init: ApiFetchInit): Promise<Response> {
  const headers = new Headers({ accept: 'application/json' });
  if (init.body !== undefined) headers.set('content-type', 'application/json');
  if (init.idempotencyKey) headers.set('idempotency-key', init.idempotencyKey);
  const request: RequestInit = {
    method: init.method ?? (init.body === undefined ? 'GET' : 'POST'),
    headers,
    credentials: 'same-origin'
  };
  if (init.body !== undefined) request.body = JSON.stringify(init.body);
  if (init.signal) request.signal = init.signal;
  return fetch(`${API_BASE}${path}`, request);
}

export async function apiFetchPayload(path: string, init: ApiFetchInit = {}): Promise<unknown> {
  const first = await sendRequest(path, init);
  const shouldRetry =
    first.status === 401 && !NO_REFRESH_PATHS.has(path) && (await refreshSession());
  const response = shouldRetry ? await sendRequest(path, init) : first;

  if (!response.ok) {
    const parsed = await readJson(response);
    throw new ApiError(response.status, isErrorResponse(parsed) ? parsed.error : FALLBACK_ERROR);
  }

  return readJson(response);
}

export async function apiFetch<T>(path: string, init: ApiFetchInit = {}): Promise<T> {
  const payload = await apiFetchPayload(path, init);
  return (isEnvelope(payload) ? payload.data : payload) as T;
}
