"use client";
import { useEffect, useState } from "react";
import { api, type BaselineSummary, type DailyRow, type HourRow, type MonthRow } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { StatCard } from "@/components/stat-card";
import { StackedAreaChart, StackedBarChart, MultiLineChart, colors } from "@/components/charts";
import { fmtAUD, fmtKWh, fmtKg, fmtPct } from "@/lib/utils";
import { Sun, Zap, ArrowDown, ArrowUp, Flame, DollarSign, Leaf, Battery } from "lucide-react";

export default function BaselinePage() {
  const [summary, setSummary] = useState<BaselineSummary | null>(null);
  const [daily, setDaily] = useState<DailyRow[]>([]);
  const [hourly, setHourly] = useState<HourRow[]>([]);
  const [monthly, setMonthly] = useState<MonthRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const [s, d, h, m] = await Promise.all([
          api.baselineSummary(),
          api.baselineDaily(),
          api.baselineHourly(),
          api.baselineMonthly(),
        ]);
        setSummary(s);
        setDaily(d);
        setHourly(h);
        setMonthly(m);
      } catch (e: any) {
        setErr(e.message);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (loading) return <div className="text-muted-foreground">Loading baseline…</div>;
  if (err) return <div className="text-destructive">Error: {err}</div>;
  if (!summary) return null;

  const formatMonth = (s: string) =>
    new Date(s).toLocaleDateString("en-AU", { month: "short", year: "2-digit" });
  const formatDay = (s: string) =>
    new Date(s).toLocaleDateString("en-AU", { day: "2-digit", month: "short" });

  return (
    <div className="space-y-8">
      <header className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Baseline</h1>
        <p className="text-muted-foreground">
          Decomposed energy flows from{" "}
          {new Date(summary.range_start).toLocaleDateString("en-AU", { day: "2-digit", month: "short", year: "numeric" })} to{" "}
          {new Date(summary.range_end).toLocaleDateString("en-AU", { day: "2-digit", month: "short", year: "numeric" })} ·{" "}
          {summary.interval_count.toLocaleString("en-AU")} 15-min intervals.
        </p>
      </header>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Annual solar yield" value={fmtKWh(summary.annual_solar_kwh)} icon={Sun} accent="solar" hint="From existing 5 kW array" />
        <StatCard label="Annual consumption" value={fmtKWh(summary.annual_consumption_kwh)} icon={Zap} accent="grid" />
        <StatCard label="Grid import" value={fmtKWh(summary.annual_import_kwh)} icon={ArrowDown} accent="negative" />
        <StatCard label="Grid export" value={fmtKWh(summary.annual_export_kwh)} icon={ArrowUp} accent="positive" />
        <StatCard label="Self-consumption" value={fmtPct(summary.self_consumption_pct)} icon={Battery} accent="battery" hint="Solar consumed on-site" />
        <StatCard label="Annual electricity cost" value={fmtAUD(summary.annual_elec_cost_aud)} icon={DollarSign} accent="primary" hint="Net of exports" />
        <StatCard label="Annual gas cost" value={fmtAUD(summary.annual_gas_cost_aud)} icon={Flame} accent="gas" hint={`${(summary.annual_gas_mj / 1000).toFixed(1)} GJ thermal`} />
        <StatCard label="Annual CO₂e" value={fmtKg(summary.annual_co2_kg)} icon={Leaf} accent="positive" hint="Grid + gas combined" />
      </section>

      <Card>
        <CardHeader>
          <CardTitle>Monthly load decomposition</CardTitle>
          <CardDescription>
            Stacked load components per month vs. solar generation. Hot water and cooking are
            shown as gas-thermal kWh equivalents (the baseline runs on gas).
          </CardDescription>
        </CardHeader>
        <CardContent>
          <StackedBarChart
            height={320}
            data={monthly}
            xKey="month"
            xFormatter={formatMonth}
            yFormatter={(v) => `${Math.round(v)} kWh`}
            series={[
              { key: "hvac_kwh", name: "HVAC", color: colors.hvac, stack: "a" },
              { key: "pool_kwh", name: "Pool pump", color: colors.pool, stack: "a" },
              { key: "base_kwh", name: "Base load", color: colors.base, stack: "a" },
              { key: "hot_water_gas_equiv_kwh", name: "Hot water (gas)", color: colors.hotwater, stack: "a" },
              { key: "cooking_gas_equiv_kwh", name: "Cooking (gas)", color: colors.cooking, stack: "a" },
            ]}
          />
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Typical day load profile</CardTitle>
            <CardDescription>Hour-of-day average load by category (×1 hour rate from 15-min intervals)</CardDescription>
          </CardHeader>
          <CardContent>
            <StackedAreaChart
              data={hourly}
              xKey="hour"
              xFormatter={(h) => `${h}:00`}
              yFormatter={(v) => `${Number(v).toFixed(2)}`}
              series={[
                { key: "hvac_kwh", name: "HVAC", color: colors.hvac, stack: "a" },
                { key: "pool_kwh", name: "Pool", color: colors.pool, stack: "a" },
                { key: "base_kwh", name: "Base", color: colors.base, stack: "a" },
                { key: "hot_water_gas_equiv_kwh", name: "Hot water", color: colors.hotwater, stack: "a" },
                { key: "cooking_gas_equiv_kwh", name: "Cooking", color: colors.cooking, stack: "a" },
                { key: "solar_kwh", name: "Solar", color: colors.solar },
              ]}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Daily flows (last 60 days)</CardTitle>
            <CardDescription>Solar, consumption and grid import / export</CardDescription>
          </CardHeader>
          <CardContent>
            <MultiLineChart
              data={daily.slice(-60)}
              xKey="day"
              xFormatter={formatDay}
              yFormatter={(v) => `${Math.round(v)}`}
              series={[
                { key: "solar_kwh", name: "Solar", color: colors.solar },
                { key: "consumption_kwh", name: "Consumption", color: colors.grid },
                { key: "grid_import_kwh", name: "Import", color: colors.negative },
                { key: "grid_export_kwh", name: "Export", color: colors.positive },
              ]}
            />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Average spot price by hour of day</CardTitle>
          <CardDescription>Used to model battery dispatch and smart-charging windows</CardDescription>
        </CardHeader>
        <CardContent>
          <MultiLineChart
            height={220}
            data={hourly}
            xKey="hour"
            xFormatter={(h) => `${h}:00`}
            yFormatter={(v) => `$${Number(v).toFixed(2)}`}
            series={[{ key: "spot_import", name: "Spot price", color: colors.ev }]}
          />
        </CardContent>
      </Card>
    </div>
  );
}
