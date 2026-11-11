import { randomUUID } from 'node:crypto';

// Synthetic data only (contract §12): no real names, passports, addresses or tokens.
export const FACILITY_CODE = 'saku-no-ie';
export const POLICY_VERSION = '2026-11-01';
export const SYNTHETIC_PASSPORT = 'TESTPP1234';

export const uid8 = () => randomUUID().replace(/-/g, '').slice(0, 8);

/** Unique synthetic booking reference, e.g. `E2E-admin-1a2b3c4d`. */
export const uniqueRef = (spec: string) => `E2E-${spec}-${uid8()}`;

/**
 * A far-future stay unique enough that a date filter isolates this run's rows
 * (year 2040–2089, day 1–28, two nights).
 */
export const uniqueStay = () => {
  const n = Number.parseInt(uid8(), 16);
  const year = 2040 + (n % 50);
  const month = 1 + (Math.floor(n / 50) % 12);
  const day = 1 + (Math.floor(n / 600) % 26);
  const pad = (v: number) => String(v).padStart(2, '0');
  return {
    checkIn: `${year}-${pad(month)}-${pad(day)}`,
    checkOut: `${year}-${pad(month)}-${pad(day + 2)}`
  };
};
