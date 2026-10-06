import type { components } from '$lib/api/schema';

export type Role = components['schemas']['Role'];
export type ProductKind = components['schemas']['ProductKind'];

const ROLE_LABELS: Record<Role, string> = { owner: 'Pemilik', courier: 'Kurir' };

const PRODUCT_KIND_LABELS: Record<ProductKind, string> = {
  refill: 'Isi ulang',
  new_gallon: 'Galon baru',
  other: 'Lainnya'
};

export const PRODUCT_KINDS: ProductKind[] = ['refill', 'new_gallon', 'other'];

export function roleLabel(role: Role): string {
  return ROLE_LABELS[role];
}

export function productKindLabel(kind: ProductKind): string {
  return PRODUCT_KIND_LABELS[kind];
}
