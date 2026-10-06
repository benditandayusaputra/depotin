import { describe, expect, it } from 'vitest';
import { maskPhone, normalizePhone } from './phone';

describe('normalizePhone', () => {
  it.each([
    ['leading 0 becomes 62 and dashes are dropped', '0812-3456-789', '628123456789'],
    ['leading +62 with spaces is kept as 62', '+62 812 3456 789', '628123456789'],
    ['leading 62 is kept as is', '62812345678', '62812345678'],
    ['leading 8 is treated as local number and prefixed with 62', '812345678', '62812345678'],
    ['dots and parentheses are ignored', '(0812) 3456.7890', '6281234567890'],
    ['landline with area code is accepted', '0215551234', '62215551234'],
    ['shortest accepted length is 9 digits', '08123456', '628123456']
  ])('%s', (_name, input, expected) => {
    expect(normalizePhone(input)).toBe(expected);
  });

  it.each([
    ['empty string', ''],
    ['letters only', 'halo'],
    ['other country code', '+1 555 123 4567'],
    ['leading 1 is not Indonesian', '1234567890'],
    ['too short after normalization', '0812'],
    ['too long after normalization', '0812345678901234']
  ])('rejects %s', (_name, input) => {
    expect(normalizePhone(input)).toBeNull();
  });
});

describe('maskPhone', () => {
  it.each([
    ['628123456789', '••••••••6789'],
    ['6789', '6789'],
    ['89', '89'],
    ['', '']
  ])('masks %s as %s', (phone, expected) => {
    expect(maskPhone(phone)).toBe(expected);
  });
});
