import { describe, expect, it } from 'vitest';
import { waUrl } from './whatsapp';

describe('waUrl', () => {
  it.each([
    ['628123456789', 'Halo', 'https://wa.me/628123456789?text=Halo'],
    [
      '628123456789',
      'Halo, pesanan #A1 & B2?',
      'https://wa.me/628123456789?text=Halo%2C%20pesanan%20%23A1%20%26%20B2%3F'
    ],
    ['628123456789', '', 'https://wa.me/628123456789?text=']
  ])('builds link for %s with text %j', (phone, text, expected) => {
    expect(waUrl(phone, text)).toBe(expected);
  });
});
