import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function fmtAUD(v: number, opts: { compact?: boolean } = {}) {
  if (v == null || Number.isNaN(v)) return "—";
  if (opts.compact && Math.abs(v) >= 1000) {
    return new Intl.NumberFormat("en-AU", {
      style: "currency",
      currency: "AUD",
      notation: "compact",
      maximumFractionDigits: 1,
    }).format(v);
  }
  return new Intl.NumberFormat("en-AU", {
    style: "currency",
    currency: "AUD",
    maximumFractionDigits: 0,
  }).format(v);
}

export function fmtKWh(v: number) {
  if (v == null || Number.isNaN(v)) return "—";
  if (Math.abs(v) >= 1000) return `${(v / 1000).toFixed(1)} MWh`;
  return `${Math.round(v).toLocaleString("en-AU")} kWh`;
}

export function fmtKg(v: number) {
  if (v == null || Number.isNaN(v)) return "—";
  if (Math.abs(v) >= 1000) return `${(v / 1000).toFixed(1)} t CO₂e`;
  return `${Math.round(v).toLocaleString("en-AU")} kg CO₂e`;
}

export function fmtYears(v: number) {
  if (v == null || Number.isNaN(v) || !isFinite(v) || v <= 0) return "—";
  return `${v.toFixed(1)} yrs`;
}

export function fmtPct(v: number) {
  if (v == null || Number.isNaN(v)) return "—";
  return `${v.toFixed(0)}%`;
}
