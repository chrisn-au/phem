// Browser-side API client. All requests go via the Next.js /api/* rewrite
// (defined in next.config.mjs) which proxies to the Go backend.

const BASE = "/api";

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text}`);
  }
  return res.json() as Promise<T>;
}

// ---- types (kept loose; mirrored from api/internal/models) ----

export type BaselineSummary = {
  range_start: string;
  range_end: string;
  interval_count: number;
  annual_solar_kwh: number;
  annual_consumption_kwh: number;
  annual_import_kwh: number;
  annual_export_kwh: number;
  annual_gas_mj: number;
  annual_elec_cost_aud: number;
  annual_gas_cost_aud: number;
  annual_co2_kg: number;
  self_consumption_pct: number;
  solar_fraction_pct: number;
};

export type DailyRow = {
  day: string;
  solar_kwh: number;
  consumption_kwh: number;
  grid_import_kwh: number;
  grid_export_kwh: number;
  battery_discharge_kwh: number;
  avg_spot_import: number;
  avg_temp_c: number;
};

export type HourRow = {
  hour: number;
  solar_kwh: number;
  consumption_kwh: number;
  hvac_kwh: number;
  pool_kwh: number;
  base_kwh: number;
  hot_water_gas_equiv_kwh: number;
  cooking_gas_equiv_kwh: number;
  ev_kwh: number;
  spot_import: number;
};

export type MonthRow = {
  month: string;
  solar_kwh: number;
  hvac_kwh: number;
  pool_kwh: number;
  base_kwh: number;
  hot_water_gas_equiv_kwh: number;
  cooking_gas_equiv_kwh: number;
  ev_kwh: number;
  grid_import_kwh: number;
  grid_export_kwh: number;
  avg_temp_c: number;
};

export type Scenario = {
  id: number;
  name: string;
  description?: string;
  upgrades: { hphws: boolean; induction: boolean; ev: boolean; solar: boolean };
  device_params: Record<string, unknown>;
  dispatch: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  result?: ScenarioResult;
};

export type ScenarioResult = {
  capex_gross_aud: number;
  capex_net_aud: number;
  annual_saving_aud: number;
  payback_years: number;
  annual_co2_saving_kg: number;
  embodied_co2_kg: number;
  carbon_payback_years: number;
  cumulative_savings: { year: number; net_saving_aud: number }[];
  breakdown: Record<string, Record<string, unknown>>;
};

export type ExploreCombo = {
  upgrades: { hphws: boolean; induction: boolean; ev: boolean; solar: boolean };
  label: string;
  capex_net_aud: number;
  annual_saving_aud: number;
  payback_years: number;
  annual_co2_saving_kg: number;
  embodied_co2_kg: number;
  carbon_payback_years: number;
  npv_20yr_aud: number;
  tags: string[] | null;
};

export type ExploreResult = {
  combos: ExploreCombo[];
  best_payback_idx: number;
  best_carbon_idx: number;
  best_npv_idx: number;
  cheapest_idx: number;
};

export type Assumption = {
  key: string;
  category: string;
  label: string;
  value: unknown;
  unit?: string;
  description?: string;
};

export type GasBill = {
  id?: number;
  period_start: string;
  period_end: string;
  consumption_mj: number;
  cost_aud?: number;
  note?: string;
};

export type IngestionRun = {
  id: number;
  source: string;
  filename?: string;
  rows_in: number;
  rows_loaded: number;
  gaps_found: number;
  duplicates: number;
  implausible: number;
  range_start?: string;
  range_end?: string;
  status: string;
  message?: string;
  created_at: string;
};

export type DataHealth = {
  total_intervals: number;
  range_start: string;
  range_end: string;
  runs: IngestionRun[];
};

// ---- endpoints ----

export const api = {
  health: () => req<{ status: string }>("/healthz"),

  baselineSummary: () => req<BaselineSummary>("/baseline/summary"),
  baselineDaily: () => req<DailyRow[]>("/baseline/daily"),
  baselineHourly: () => req<HourRow[]>("/baseline/hourly"),
  baselineMonthly: () => req<MonthRow[]>("/baseline/monthly"),

  dataHealth: () => req<DataHealth>("/data/health"),
  listGasBills: () => req<GasBill[]>("/data/gas-bills"),
  createGasBill: (b: GasBill) =>
    req<GasBill>("/data/gas-bills", { method: "POST", body: JSON.stringify(b) }),

  listScenarios: () => req<Scenario[]>("/scenarios"),
  upsertScenario: (s: Partial<Scenario>) =>
    req<Scenario>("/scenarios", { method: "POST", body: JSON.stringify(s) }),
  deleteScenario: (id: number) =>
    req<{ ok: boolean }>(`/scenarios/${id}`, { method: "DELETE" }),
  computeScenario: (id: number) =>
    req<ScenarioResult>(`/scenarios/${id}/compute`, { method: "POST" }),
  recomputeAll: () =>
    req<{ ok: boolean }>("/scenarios/recompute-all", { method: "POST" }),
  exploreScenarios: () => req<ExploreResult>("/scenarios/explore"),

  listAssumptions: () => req<Assumption[]>("/assumptions"),
  updateAssumption: (key: string, value: unknown) =>
    req<{ key: string; value: unknown }>(`/assumptions/${encodeURIComponent(key)}`, {
      method: "PUT",
      body: JSON.stringify({ value }),
    }),
};

// CSV download URL
export const exportScenariosCSV = `${BASE}/scenarios/export.csv`;

// File upload helpers (multipart)
export async function uploadCSV(kind: "solar" | "consumption", file: File) {
  const fd = new FormData();
  fd.append("file", file);
  const res = await fetch(`${BASE}/data/upload/${kind}`, { method: "POST", body: fd });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}
