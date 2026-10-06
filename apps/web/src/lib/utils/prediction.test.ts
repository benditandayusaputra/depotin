import { describe, expect, it } from 'vitest';
import { formatDays, predictionSentence } from './prediction';

describe('predictionSentence', () => {
  it('uses a decimal comma and the sample count', () => {
    expect(
      predictionSentence({
        days_per_gallon: 3.5,
        prediction_samples: 6,
        prediction_confidence: 'high'
      })
    ).toBe('Biasanya 1 galon habis dalam 3,5 hari, dihitung dari 6 pesanan terakhir');
  });

  it('explains the depot default when there are no samples', () => {
    expect(
      predictionSentence({
        days_per_gallon: 4,
        prediction_samples: 0,
        prediction_confidence: 'low'
      })
    ).toBe('Biasanya 1 galon habis dalam 4 hari, memakai angka bawaan depot');
  });

  it('has no prediction without deliveries', () => {
    expect(
      predictionSentence({
        days_per_gallon: null,
        prediction_samples: 0,
        prediction_confidence: 'none'
      })
    ).toBe('Belum ada prediksi');
    expect(formatDays(2.25)).toBe('2,3 hari');
  });
});
