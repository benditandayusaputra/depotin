import type { components } from '$lib/api/schema';

export type OrderStatus = components['schemas']['OrderStatus'];
export type PaymentStatus = components['schemas']['TrackedOrder']['payment_status'];

const STATUS_LABELS: Record<OrderStatus, string> = {
  pending: 'Menunggu konfirmasi',
  confirmed: 'Dikonfirmasi',
  on_delivery: 'Sedang diantar',
  delivered: 'Selesai',
  cancelled: 'Dibatalkan'
};

const STATUS_CLASSES: Record<OrderStatus, string> = {
  pending: 'bg-status-pending-soft text-status-pending',
  confirmed: 'bg-status-confirmed-soft text-status-confirmed',
  on_delivery: 'bg-status-delivery-soft text-status-delivery',
  delivered: 'bg-status-done-soft text-status-done',
  cancelled: 'bg-status-cancelled-soft text-status-cancelled'
};

export function statusLabel(status: OrderStatus): string {
  return STATUS_LABELS[status];
}

export function statusClasses(status: OrderStatus): string {
  return STATUS_CLASSES[status];
}

export function isFinalStatus(status: OrderStatus): boolean {
  return status === 'delivered' || status === 'cancelled';
}

export function paymentLabel(status: PaymentStatus): string {
  return status === 'paid' ? 'Lunas' : 'Belum bayar';
}
