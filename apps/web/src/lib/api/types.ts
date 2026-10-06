import type { components } from './schema';

export type Order = components['schemas']['Order'];
export type OrderCreate = components['schemas']['OrderCreate'];
export type Customer = components['schemas']['Customer'];
export type CustomerCreate = components['schemas']['CustomerCreate'];
export type CustomerUpdate = components['schemas']['CustomerUpdate'];
export type LedgerEntry = components['schemas']['LedgerEntry'];
export type Reminder = components['schemas']['Reminder'];
export type Product = components['schemas']['Product'];
export type DashboardToday = components['schemas']['DashboardToday'];
export type ReportSummary = components['schemas']['ReportSummary'];
export type GallonSummary = components['schemas']['GallonSummary'];
export type LinkInfo = { link: string; wa_url: string };
