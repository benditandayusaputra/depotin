const rupiahDigits = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 });

export function formatRupiah(amount: number): string {
  const sign = amount < 0 ? '-' : '';
  return `${sign}Rp${rupiahDigits.format(Math.abs(amount))}`;
}
