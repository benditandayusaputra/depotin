import { ApiError } from '$lib/api/client';

export const GENERIC_ERROR_MESSAGE = 'Terjadi gangguan. Coba lagi.';

export function errorMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : GENERIC_ERROR_MESSAGE;
}

export function fieldErrors(error: unknown): Record<string, string> {
  return error instanceof ApiError ? error.fields : {};
}

export function hasFieldErrors(errors: Record<string, string>): boolean {
  return Object.keys(errors).length > 0;
}
