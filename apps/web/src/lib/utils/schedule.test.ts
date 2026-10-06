import { describe, expect, it } from 'vitest';
import { jakartaDate, scheduledDayLabel } from './schedule';

const now = new Date('2026-10-06T20:00:00Z');

describe('jakartaDate', () => {
  it('uses the Jakarta calendar day', () => {
    expect(jakartaDate(0, now)).toBe('2026-10-07');
    expect(jakartaDate(1, now)).toBe('2026-10-08');
  });
});

describe('scheduledDayLabel', () => {
  it('labels today, tomorrow and other days', () => {
    expect(scheduledDayLabel('2026-10-07', now)).toBe('hari ini');
    expect(scheduledDayLabel('2026-10-08', now)).toBe('besok');
    expect(scheduledDayLabel('2026-10-10', now)).toContain('10 Okt 2026');
  });
});
