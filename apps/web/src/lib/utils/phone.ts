const COUNTRY_CODE = '62';
const MIN_DIGITS = 9;
const MAX_DIGITS = 15;
const MASK_CHAR = '•';
const VISIBLE_TAIL = 4;

export function normalizePhone(input: string): string | null {
  const digits = input.replace(/\D/g, '');
  let normalized: string;
  if (digits.startsWith('0')) normalized = COUNTRY_CODE + digits.slice(1);
  else if (digits.startsWith(COUNTRY_CODE)) normalized = digits;
  else if (digits.startsWith('8')) normalized = COUNTRY_CODE + digits;
  else return null;
  if (normalized.length < MIN_DIGITS || normalized.length > MAX_DIGITS) return null;
  return normalized;
}

export function maskPhone(phone: string): string {
  const hiddenLength = Math.max(phone.length - VISIBLE_TAIL, 0);
  return MASK_CHAR.repeat(hiddenLength) + phone.slice(hiddenLength);
}
