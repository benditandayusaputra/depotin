import type { components } from '$lib/api/schema';

export type Role = components['schemas']['Role'];
export type ProductKind = components['schemas']['ProductKind'];

const ROLE_LABELS: Record<Role, string> = { owner: 'Pemilik', courier: 'Kurir' };

const PRODUCT_KIND_LABELS: Record<ProductKind, string> = {
  refill: 'Isi ulang',
  new_gallon: 'Galon baru',
  other: 'Lainnya'
};

export const PRODUCT_KINDS: ProductKind[] = ['refill', 'new_gallon', 'other'];

export function roleLabel(role: Role): string {
  return ROLE_LABELS[role];
}

export function productKindLabel(kind: ProductKind): string {
  return PRODUCT_KIND_LABELS[kind];
}

export type OrderStatus = components['schemas']['OrderStatus'];
export type OrderSource = components['schemas']['OrderSource'];
export type Confidence = components['schemas']['Confidence'];
export type PaymentMethod = components['schemas']['PaymentMethod'];
export type Fulfilment = components['schemas']['Fulfilment'];

const ORDER_STATUS_LABELS: Record<OrderStatus, string> = {
  pending: 'Menunggu konfirmasi',
  confirmed: 'Dikonfirmasi',
  on_delivery: 'Sedang diantar',
  delivered: 'Selesai',
  cancelled: 'Dibatalkan'
};

const ORDER_SOURCE_LABELS: Record<OrderSource, string> = {
  public: 'Halaman depot',
  link: 'Link pribadi',
  reminder: 'Pengingat',
  owner: 'Pemilik',
  courier: 'Kurir'
};

const CONFIDENCE_LABELS: Record<Confidence, string> = {
  none: 'Belum ada',
  low: 'Rendah',
  medium: 'Sedang',
  high: 'Tinggi'
};

const PAYMENT_METHOD_LABELS: Record<PaymentMethod, string> = {
  cash: 'Tunai',
  transfer: 'Transfer',
  qris: 'QRIS'
};

const FULFILMENT_LABELS: Record<Fulfilment, string> = {
  delivery: 'Diantar',
  pickup: 'Ambil sendiri'
};

export const ORDER_STATUSES: OrderStatus[] = [
  'pending',
  'confirmed',
  'on_delivery',
  'delivered',
  'cancelled'
];

export const PAYMENT_METHODS: PaymentMethod[] = ['cash', 'transfer', 'qris'];

export function orderStatusLabel(status: OrderStatus): string {
  return ORDER_STATUS_LABELS[status];
}

export function orderSourceLabel(source: OrderSource): string {
  return ORDER_SOURCE_LABELS[source];
}

export function confidenceLabel(confidence: Confidence): string {
  return CONFIDENCE_LABELS[confidence];
}

export function paymentMethodLabel(method: PaymentMethod): string {
  return PAYMENT_METHOD_LABELS[method];
}

export function fulfilmentLabel(fulfilment: Fulfilment): string {
  return FULFILMENT_LABELS[fulfilment];
}

export type PaymentStatus = components['schemas']['Order']['payment_status'];

export function isFinalStatus(status: OrderStatus): boolean {
  return status === 'delivered' || status === 'cancelled';
}

export function paymentStatusLabel(status: PaymentStatus): string {
  return status === 'paid' ? 'Lunas' : 'Belum bayar';
}
