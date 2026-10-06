import { describe, expect, it } from 'vitest';
import {
  ORDER_STATUSES,
  PRODUCT_KINDS,
  confidenceLabel,
  orderSourceLabel,
  orderStatusLabel,
  productKindLabel,
  roleLabel
} from './labels';

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

describe('order labels', () => {
  it('maps every order status to an Indonesian label', () => {
    expect(ORDER_STATUSES.map(orderStatusLabel)).toEqual([
      'Menunggu konfirmasi',
      'Dikonfirmasi',
      'Sedang diantar',
      'Selesai',
      'Dibatalkan'
    ]);
  });

  it('maps sources and confidence levels', () => {
    expect(orderSourceLabel('reminder')).toBe('Pengingat');
    expect(confidenceLabel('none')).toBe('Belum ada');
    expect(confidenceLabel('high')).toBe('Tinggi');
  });
});
