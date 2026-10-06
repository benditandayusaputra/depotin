export type ToastKind = 'success' | 'error' | 'info';

export interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

const TOAST_DURATION_MS = 4000;

let nextId = 0;

export const toasts = $state<Toast[]>([]);

export function dismissToast(id: number): void {
  const index = toasts.findIndex((toast) => toast.id === id);
  if (index !== -1) toasts.splice(index, 1);
}

export function pushToast(kind: ToastKind, message: string): number {
  const id = ++nextId;
  toasts.push({ id, kind, message });
  setTimeout(() => dismissToast(id), TOAST_DURATION_MS);
  return id;
}
