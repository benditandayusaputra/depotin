import type { components } from '$lib/api/schema';

type Customer = components['schemas']['Customer'];
type PredictionFields = Pick<
  Customer,
  'days_per_gallon' | 'prediction_samples' | 'prediction_confidence'
>;

const dayFormatter = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 1 });

export function formatDays(days: number): string {
  return `${dayFormatter.format(days)} hari`;
}

export function predictionSentence(customer: PredictionFields): string {
  if (customer.prediction_confidence === 'none' || customer.days_per_gallon === null) {
    return 'Belum ada prediksi';
  }
  const base = `Biasanya 1 galon habis dalam ${formatDays(customer.days_per_gallon)}`;
  if (customer.prediction_samples === 0) return `${base}, memakai angka bawaan depot`;
  return `${base}, dihitung dari ${customer.prediction_samples} pesanan terakhir`;
}
