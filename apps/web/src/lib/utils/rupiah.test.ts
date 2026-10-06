import { describe, expect, it } from 'vitest';
import { formatRupiah } from './rupiah';

describe('formatRupiah', () => {
  it.each([
    [0, 'Rp0'],
    [500, 'Rp500'],
    [12000, 'Rp12.000'],
    [1250000, 'Rp1.250.000'],
    [12000.4, 'Rp12.000'],
    [12000.6, 'Rp12.001'],
    [-7000, '-Rp7.000']
  ])('formats %d as %s', (amount, expected) => {
    expect(formatRupiah(amount)).toBe(expected);
  });
});
