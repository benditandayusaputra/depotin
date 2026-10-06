import { describe, expect, it } from 'vitest';
import { ApiError } from '$lib/api/client';
import { GENERIC_ERROR_MESSAGE, errorMessage, fieldErrors, hasFieldErrors } from './errors';

const validation = new ApiError(422, {
  code: 'validation_failed',
  message: 'Data belum valid.',
  fields: { phone: 'Minimal 9 karakter.' }
});

describe('errorMessage', () => {
  it('uses the API message for ApiError', () => {
    expect(errorMessage(validation)).toBe('Data belum valid.');
  });

  it('falls back to a generic message for other errors', () => {
    expect(errorMessage(new TypeError('fetch failed'))).toBe(GENERIC_ERROR_MESSAGE);
    expect(errorMessage(undefined)).toBe(GENERIC_ERROR_MESSAGE);
  });
});

describe('fieldErrors', () => {
  it('returns the fields of an ApiError', () => {
    expect(fieldErrors(validation)).toEqual({ phone: 'Minimal 9 karakter.' });
  });

  it('returns an empty object for ApiError without fields and for other errors', () => {
    expect(fieldErrors(new ApiError(401, { code: 'unauthenticated', message: 'x' }))).toEqual({});
    expect(fieldErrors(new Error('x'))).toEqual({});
  });
});

describe('hasFieldErrors', () => {
  it('detects presence of field errors', () => {
    expect(hasFieldErrors({})).toBe(false);
    expect(hasFieldErrors({ name: 'Wajib diisi.' })).toBe(true);
  });
});
