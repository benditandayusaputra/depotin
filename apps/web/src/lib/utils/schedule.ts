import { formatDate } from './date';

const DAY_MS = 24 * 60 * 60 * 1000;

const jakartaIsoDate = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Asia/Jakarta',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit'
});

export function jakartaDate(daysFromNow: number, now: Date = new Date()): string {
  return jakartaIsoDate.format(new Date(now.getTime() + daysFromNow * DAY_MS));
}

export function scheduledDayLabel(date: string, now: Date = new Date()): string {
  if (date === jakartaDate(0, now)) return 'hari ini';
  if (date === jakartaDate(1, now)) return 'besok';
  return formatDate(date);
}
