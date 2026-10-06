import type { components } from '$lib/api/schema';
import { serverApi } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const prerender = false;

export const load: PageServerLoad = async (event) => {
  const page = await serverApi<components['schemas']['PublicDepotPage']>(
    event,
    `/public/depots/${encodeURIComponent(event.params.slug)}`
  );
  event.setHeaders({ 'cache-control': 'public, s-maxage=60, stale-while-revalidate=300' });
  return page;
};
