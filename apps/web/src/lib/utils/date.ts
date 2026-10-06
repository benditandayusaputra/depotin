const TIME_ZONE = 'Asia/Jakarta';
const JAKARTA_OFFSET_MS = 7 * 60 * 60 * 1000;
const DAY_MS = 24 * 60 * 60 * 1000;

const dateFormatter = new Intl.DateTimeFormat('id-ID', {
  weekday: 'short',
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  timeZone: TIME_ZONE
});

const timeFormatter = new Intl.DateTimeFormat('id-ID', {
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
  timeZone: TIME_ZONE
});

export function formatDate(iso: string): string {
  return dateFormatter.format(new Date(iso));
}

export function formatTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

function jakartaDayIndex(date: Date): number {
  return Math.floor((date.getTime() + JAKARTA_OFFSET_MS) / DAY_MS);
}

export function relativeDays(iso: string, now: Date = new Date()): string {
  const diff = jakartaDayIndex(new Date(iso)) - jakartaDayIndex(now);
  if (diff === 0) return 'hari ini';
  if (diff === 1) return 'besok';
  if (diff === -1) return 'kemarin';
  return diff > 0 ? `${diff} hari lagi` : `${-diff} hari lalu`;
}

const isoDateFormatter = new Intl.DateTimeFormat('en-CA', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  timeZone: TIME_ZONE
});

const dayFormatter = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'short',
  timeZone: TIME_ZONE
});

export function isoDate(date: Date = new Date()): string {
  return isoDateFormatter.format(date);
}

export function shiftDays(iso: string, days: number): string {
  return isoDate(new Date(new Date(`${iso}T12:00:00+07:00`).getTime() + days * DAY_MS));
}

export function formatDateTime(iso: string): string {
  return `${formatDate(iso)} ${formatTime(iso)}`;
}

export function formatShortDay(iso: string): string {
  return dayFormatter.format(new Date(iso));
}
