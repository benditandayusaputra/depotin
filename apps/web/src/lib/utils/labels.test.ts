import { describe, expect, it } from 'vitest';
import { PRODUCT_KINDS, productKindLabel, roleLabel } from './labels';

describe('roleLabel', () => {
  it('maps roles to Indonesian labels', () => {
    expect(roleLabel('owner')).toBe('Pemilik');
    expect(roleLabel('courier')).toBe('Kurir');
  });
});

describe('productKindLabel', () => {
  it('maps every product kind to an Indonesian label', () => {
    expect(PRODUCT_KINDS.map(productKindLabel)).toEqual(['Isi ulang', 'Galon baru', 'Lainnya']);
  });
});
