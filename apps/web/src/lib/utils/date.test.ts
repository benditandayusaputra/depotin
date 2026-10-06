import { describe, expect, it } from 'vitest';
import { formatDate, formatTime, relativeDays } from './date';

describe('formatDate', () => {
  it.each([
    ['2026-10-06T03:00:00Z', 'Sel, 6 Okt 2026'],
    ['2026-10-05T18:00:00Z', 'Sel, 6 Okt 2026'],
    ['2026-01-01T16:59:59Z', 'Kam, 1 Jan 2026'],
    ['2026-01-01T17:00:00Z', 'Jum, 2 Jan 2026']
  ])('formats %s in Asia/Jakarta as %s', (iso, expected) => {
    expect(formatDate(iso)).toBe(expected);
  });
});

describe('formatTime', () => {
  it.each([
    ['2026-10-06T03:05:00Z', '10.05'],
    ['2026-10-06T17:00:00Z', '00.00'],
    ['2026-10-06T16:59:00Z', '23.59']
  ])('formats %s in Asia/Jakarta as %s', (iso, expected) => {
    expect(formatTime(iso)).toBe(expected);
  });
});

describe('relativeDays', () => {
  const now = new Date('2026-10-06T05:00:00Z');

  it.each([
    ['same Jakarta day', '2026-10-06T10:00:00Z', 'hari ini'],
    ['just after Jakarta midnight counts as today', '2026-10-05T17:00:00Z', 'hari ini'],
    ['just before Jakarta midnight counts as yesterday', '2026-10-05T16:59:59Z', 'kemarin'],
    ['next day', '2026-10-07T01:00:00Z', 'besok'],
    ['three days ahead', '2026-10-09T01:00:00Z', '3 hari lagi'],
    ['three days ago', '2026-10-03T01:00:00Z', '3 hari lalu']
  ])('%s', (_name, iso, expected) => {
    expect(relativeDays(iso, now)).toBe(expected);
  });
});
