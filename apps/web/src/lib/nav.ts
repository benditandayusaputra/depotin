import type { Pathname } from '$app/types';

export interface NavItem {
  href: Pathname;
  label: string;
  icon: string;
}

export const ICONS = {
  home: 'M3 11l9-8 9 8v9a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z',
  orders:
    'M9 3h6a1 1 0 0 1 1 1v1H8V4a1 1 0 0 1 1-1zM6 5h12a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1zM9 11h6M9 15h4',
  customers:
    'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75',
  bell: 'M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 0 1-3.46 0',
  more: 'M4 12a1.5 1.5 0 1 0 3 0 1.5 1.5 0 1 0-3 0M10.5 12a1.5 1.5 0 1 0 3 0 1.5 1.5 0 1 0-3 0M17 12a1.5 1.5 0 1 0 3 0 1.5 1.5 0 1 0-3 0',
  gallon: 'M12 2.5c-3.6 4.6-7 8.6-7 12.3a7 7 0 0 0 14 0c0-3.7-3.4-7.7-7-12.3Z',
  report: 'M18 20V10M12 20V4M6 20v-6',
  settings:
    'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z',
  logout: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9'
} as const;

export const PRIMARY_NAV: NavItem[] = [
  { href: '/app', label: 'Hari ini', icon: ICONS.home },
  { href: '/app/pesanan', label: 'Pesanan', icon: ICONS.orders },
  { href: '/app/pelanggan', label: 'Pelanggan', icon: ICONS.customers },
  { href: '/app/pengingat', label: 'Pengingat', icon: ICONS.bell }
];

export const SECONDARY_NAV: NavItem[] = [
  { href: '/app/galon', label: 'Galon', icon: ICONS.gallon },
  { href: '/app/laporan', label: 'Laporan', icon: ICONS.report },
  { href: '/app/pengaturan', label: 'Pengaturan', icon: ICONS.settings }
];

export const MORE_NAV: NavItem = { href: '/app/lainnya', label: 'Lainnya', icon: ICONS.more };

export function isActivePath(pathname: string, href: Pathname): boolean {
  if (href === '/app') return pathname === href;
  if (href === MORE_NAV.href) {
    return [MORE_NAV, ...SECONDARY_NAV].some((item) => pathname.startsWith(item.href));
  }
  return pathname.startsWith(href);
}
