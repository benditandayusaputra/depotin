import type { components } from '$lib/api/schema';
import { serverApi } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
  const me = await serverApi<components['schemas']['PersonalPage']>(
    event,
    `/public/me/${encodeURIComponent(event.params.token)}`
  );
  event.setHeaders({
    'cache-control': 'no-store',
    'x-robots-tag': 'noindex',
    'referrer-policy': 'no-referrer'
  });
  return { me, token: event.params.token, reminder: event.url.searchParams.get('r') };
};
