import type { components } from '$lib/api/schema';
import { serverApi } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
  const order = await serverApi<components['schemas']['TrackedOrder']>(
    event,
    `/public/track/${encodeURIComponent(event.params.token)}`
  );
  event.setHeaders({
    'cache-control': 'no-store',
    'x-robots-tag': 'noindex',
    'referrer-policy': 'no-referrer'
  });
  return { order, token: event.params.token };
};
