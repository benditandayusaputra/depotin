import { env } from '$env/dynamic/private';
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const FORWARDED_REQUEST_HEADERS = [
  'content-type',
  'cookie',
  'idempotency-key',
  'origin',
  'user-agent',
  'accept',
  'x-request-id'
];
const FORWARDED_RESPONSE_HEADERS = [
  'content-type',
  'cache-control',
  'etag',
  'x-request-id',
  'retry-after'
];
const UPSTREAM_TIMEOUT_MS = 15_000;

type StreamingRequestInit = RequestInit & { duplex?: 'half' };

function errorResponse(status: number, code: string, message: string): Response {
  return json({ error: { code, message } }, { status });
}

function notFound(): Response {
  return errorResponse(404, 'not_found', 'Alamat tidak ditemukan.');
}

function buildUpstreamHeaders(request: Request, clientAddress: string): Headers {
  const headers = new Headers();
  for (const name of FORWARDED_REQUEST_HEADERS) {
    const value = request.headers.get(name);
    if (value !== null) headers.set(name, value);
  }
  if (!headers.has('x-request-id')) headers.set('x-request-id', crypto.randomUUID());
  if (env.EDGE_KEY) headers.set('x-edge-key', env.EDGE_KEY);
  headers.set('x-client-ip', clientAddress);
  return headers;
}

function buildDownstreamHeaders(upstream: Response): Headers {
  const headers = new Headers();
  for (const name of FORWARDED_RESPONSE_HEADERS) {
    const value = upstream.headers.get(name);
    if (value !== null) headers.set(name, value);
  }
  for (const cookie of upstream.headers.getSetCookie()) headers.append('set-cookie', cookie);
  return headers;
}

const handler: RequestHandler = async ({ params, request, url, getClientAddress }) => {
  if (!params.path.startsWith('v1/')) return notFound();
  if (request.method === 'GET' && params.path === 'v1/stream') return notFound();

  const apiOrigin = env.API_ORIGIN ?? 'http://localhost:8080';
  const target = new URL(`/api/${params.path}${url.search}`, apiOrigin);
  const init: StreamingRequestInit = {
    method: request.method,
    headers: buildUpstreamHeaders(request, getClientAddress()),
    signal: AbortSignal.timeout(UPSTREAM_TIMEOUT_MS)
  };
  if (request.method !== 'GET' && request.method !== 'HEAD') {
    init.body = request.body;
    init.duplex = 'half';
  }

  let upstream: Response;
  try {
    upstream = await fetch(target, init);
  } catch {
    return errorResponse(
      502,
      'unavailable',
      'Server sedang tidak bisa dihubungi. Coba lagi sebentar.'
    );
  }

  return new Response(upstream.body, {
    status: upstream.status,
    headers: buildDownstreamHeaders(upstream)
  });
};

export const GET = handler;
export const POST = handler;
export const PATCH = handler;
export const PUT = handler;
export const DELETE = handler;
