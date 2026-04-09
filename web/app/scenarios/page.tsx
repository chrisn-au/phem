"use client";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { api, exportScenariosCSV, type ExploreCombo, type ExploreResult, type Scenario } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Slider } from "@/components/ui/slider";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { MultiLineChart, colors } from "@/components/charts";
import { fmtAUD, fmtKg, fmtYears } from "@/lib/utils";
import {
  Award,
  Car,
  ChefHat,
  Compass,
  DollarSign,
  Download,
  Flame,
  GitCompare,
  Leaf,
  Plus,
  RefreshCw,
  Sparkles,
  Sun,
  Trash2,
  Wrench,
  Zap,
} from "lucide-react";

type Upgrades = Scenario["upgrades"];

export default function ScenariosPage() {
  const [scenarios, setScenarios] = useState<Scenario[]>([]);
  const [explore, setExplore] = useState<ExploreResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [exploring, setExploring] = useState(false);
  const [activeId, setActiveId] = useState<number | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const refresh = async () => {
    const list = await api.listScenarios();
    setScenarios(list);
    if (list.length && activeId == null) setActiveId(list[0].id);
    setLoading(false);
  };

  useEffect(() => {
    refresh().catch((e) => toast.error(e.message));
    api.exploreScenarios().then(setExplore).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const active = scenarios.find((s) => s.id === activeId) || null;

  const upsert = async (s: Partial<Scenario>) => {
    const updated = await api.upsertScenario(s);
    toast.success(`Saved ${updated.name}`);
    await refresh();
    setActiveId(updated.id);
    return updated;
  };

  const updateActive = async (mut: (s: Scenario) => void) => {
    if (!active) return;
    const next: Scenario = JSON.parse(JSON.stringify(active));
    mut(next);
    await upsert(next);
  };

  const remove = async (id: number) => {
    await api.deleteScenario(id);
    toast.success("Deleted");
    await refresh();
    if (activeId === id) setActiveId(null);
  };

  const recompute = async () => {
    await api.recomputeAll();
    await api.exploreScenarios().then(setExplore);
    toast.success("Recomputed all");
    await refresh();
  };

  const runExplore = async () => {
    setExploring(true);
    try {
      const r = await api.exploreScenarios();
      setExplore(r);
      toast.success("Explored 16 combinations");
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      setExploring(false);
    }
  };

  const saveCombo = async (c: ExploreCombo) => {
    const name = window.prompt("Save this combination as a scenario:", c.label);
    if (!name) return;
    await upsert({
      name,
      description: "Saved from Explore",
      upgrades: c.upgrades,
      device_params: {
        hphws_cop: 3.5,
        induction_eff_ratio: 3.0,
        ev_kwh_per_100km: 16,
        ev_include_vehicle: false,
        solar_panel: "premium",
        solar_panel_count: 15,
      },
      dispatch: {},
    } as Partial<Scenario>);
  };

  if (loading) return <div className="text-muted-foreground">Loading scenarios…</div>;

  // Build merged comparison chart data
  const horizon = scenarios[0]?.result?.cumulative_savings.length || 21;
  const chartData = Array.from({ length: horizon }, (_, y) => {
    const point: Record<string, number> = { year: y };
    scenarios.forEach((s) => {
      if (s.result?.cumulative_savings[y]) {
        point[s.name] = s.result.cumulative_savings[y].net_saving_aud;
      }
    });
    return point;
  });
  const palette = [colors.solar, colors.battery, colors.ev, colors.hvac, colors.gas, colors.cooking, colors.pool, colors.hotwater];
  const lineSeries = scenarios.map((s, i) => ({ key: s.name, name: s.name, color: palette[i % palette.length] }));

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Scenarios</h1>
          <p className="text-muted-foreground">
            Compare upgrade combinations side-by-side. Use <strong>Explore</strong> to evaluate
            all 16 possibilities at once and save the winners.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={recompute}>
            <RefreshCw className="mr-2 h-4 w-4" /> Recompute all
          </Button>
          <Button asChild variant="outline">
            <a href={exportScenariosCSV} download>
              <Download className="mr-2 h-4 w-4" /> Export CSV
            </a>
          </Button>
          <Button onClick={() => setDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" /> New scenario
          </Button>
        </div>
      </header>

      {/* ---- Explore + recommendations ---- */}
      <Card className="border-primary/30 bg-gradient-to-br from-primary/5 to-transparent">
        <CardHeader className="flex-row items-start justify-between gap-4 space-y-0">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-primary" /> Smart explorer
            </CardTitle>
            <CardDescription>
              Brute-force all 16 upgrade combinations against your baseline data. Tagged winners
              are the cheapest entry, fastest payback, biggest carbon cut, and best 20-year value.
            </CardDescription>
          </div>
          <Button variant="default" onClick={runExplore} disabled={exploring}>
            <Compass className="mr-2 h-4 w-4" />
            {exploring ? "Running…" : explore ? "Re-explore" : "Run explore"}
          </Button>
        </CardHeader>
        {explore && (
          <CardContent>
            <RecommendationCards explore={explore} onSave={saveCombo} />
            <div className="mt-6 overflow-x-auto rounded-md border bg-card">
              <table className="w-full text-sm">
                <thead className="bg-muted/50">
                  <tr className="text-left">
                    <th className="px-3 py-2 font-medium">Combination</th>
                    <th className="px-3 py-2 font-medium text-right">Capex (net)</th>
                    <th className="px-3 py-2 font-medium text-right">$/yr saving</th>
                    <th className="px-3 py-2 font-medium text-right">Payback</th>
                    <th className="px-3 py-2 font-medium text-right">CO₂/yr</th>
                    <th className="px-3 py-2 font-medium text-right">20-yr NPV</th>
                    <th className="px-3 py-2 font-medium text-right"></th>
                  </tr>
                </thead>
                <tbody>
                  {[...explore.combos]
                    .sort((a, b) => (b.npv_20yr_aud || 0) - (a.npv_20yr_aud || 0))
                    .map((c, i) => (
                      <tr key={i} className="border-t hover:bg-accent/30">
                        <td className="px-3 py-2">
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{c.label}</span>
                            <UpgradeIcons upgrades={c.upgrades} />
                            {c.tags?.map((t) => (
                              <Badge key={t} variant={t === "best_20yr_value" ? "default" : t === "best_carbon" ? "success" : t === "best_payback" ? "warning" : "outline"} className="text-[10px]">
                                {tagLabels[t] || t}
                              </Badge>
                            ))}
                          </div>
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums">{fmtAUD(c.capex_net_aud)}</td>
                        <td className="px-3 py-2 text-right tabular-nums text-positive">{fmtAUD(c.annual_saving_aud)}</td>
                        <td className="px-3 py-2 text-right tabular-nums">{fmtYears(c.payback_years)}</td>
                        <td className="px-3 py-2 text-right tabular-nums">{fmtKg(c.annual_co2_saving_kg)}</td>
                        <td className={`px-3 py-2 text-right tabular-nums ${c.npv_20yr_aud >= 0 ? "text-positive" : "text-destructive"}`}>{fmtAUD(c.npv_20yr_aud, { compact: true })}</td>
                        <td className="px-3 py-2 text-right">
                          <Button size="sm" variant="ghost" onClick={() => saveCombo(c)}>
                            <Plus className="h-4 w-4" />
                          </Button>
                        </td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        )}
      </Card>

      {/* ---- Comparison table ---- */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <GitCompare className="h-5 w-5 text-primary" /> Saved scenarios
          </CardTitle>
          <CardDescription>Click a row to configure it. Changes save and recompute on toggle.</CardDescription>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="border-b text-muted-foreground">
              <tr className="text-left">
                <th className="py-2 font-medium">Scenario</th>
                <th className="py-2 font-medium text-center">Upgrades</th>
                <th className="py-2 font-medium text-right">Capex (net)</th>
                <th className="py-2 font-medium text-right">Annual saving</th>
                <th className="py-2 font-medium text-right">Payback</th>
                <th className="py-2 font-medium text-right">Annual CO₂ saved</th>
                <th className="py-2 font-medium text-right">Carbon payback</th>
                <th className="py-2 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {scenarios.map((s) => (
                <tr
                  key={s.id}
                  className={`cursor-pointer border-b transition-colors hover:bg-accent/50 ${activeId === s.id ? "bg-accent/30" : ""}`}
                  onClick={() => setActiveId(s.id)}
                >
                  <td className="py-3">
                    <div className="font-medium">{s.name}</div>
                    {s.description && <div className="text-xs text-muted-foreground">{s.description}</div>}
                  </td>
                  <td className="py-3">
                    <div className="flex justify-center">
                      <UpgradeIcons upgrades={s.upgrades} />
                    </div>
                  </td>
                  <td className="py-3 text-right tabular-nums">{fmtAUD(s.result?.capex_net_aud || 0)}</td>
                  <td className="py-3 text-right tabular-nums text-positive">{fmtAUD(s.result?.annual_saving_aud || 0)}</td>
                  <td className="py-3 text-right tabular-nums">{fmtYears(s.result?.payback_years || 0)}</td>
                  <td className="py-3 text-right tabular-nums">{fmtKg(s.result?.annual_co2_saving_kg || 0)}</td>
                  <td className="py-3 text-right tabular-nums">{fmtYears(s.result?.carbon_payback_years || 0)}</td>
                  <td className="py-3">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={(e) => {
                        e.stopPropagation();
                        remove(s.id);
                      }}
                    >
                      <Trash2 className="h-4 w-4 text-muted-foreground" />
                    </Button>
                  </td>
                </tr>
              ))}
              {scenarios.length === 0 && (
                <tr>
                  <td colSpan={8} className="py-6 text-center text-muted-foreground">
                    No scenarios yet — use Explore above or click <strong>New scenario</strong>.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>20-year cumulative net saving</CardTitle>
          <CardDescription>
            Negative values are upfront capex; the line crosses zero at simple payback.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <MultiLineChart
            height={360}
            data={chartData}
            xKey="year"
            xFormatter={(v) => `Y${v}`}
            yFormatter={(v) => fmtAUD(Number(v), { compact: true })}
            zeroLine
            series={lineSeries}
          />
        </CardContent>
      </Card>

      {active && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wrench className="h-5 w-5 text-primary" /> {active.name} — configure
            </CardTitle>
            <CardDescription>Toggle upgrades and tweak parameters. Saves automatically.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
              <ToggleCard
                icon={<Flame className="h-4 w-4 text-hotwater" />}
                title="Heat pump hot water"
                description="Replace gas instant HW"
                checked={active.upgrades.hphws}
                onChange={(c) => updateActive((s) => (s.upgrades.hphws = c))}
              />
              <ToggleCard
                icon={<ChefHat className="h-4 w-4 text-cooking" />}
                title="Induction cooktop"
                description="Replace gas cooktop"
                checked={active.upgrades.induction}
                onChange={(c) => updateActive((s) => (s.upgrades.induction = c))}
              />
              <ToggleCard
                icon={<Car className="h-4 w-4 text-ev" />}
                title="Electric vehicle"
                description="Replace petrol CX5"
                checked={active.upgrades.ev}
                onChange={(c) => updateActive((s) => (s.upgrades.ev = c))}
              />
              <ToggleCard
                icon={<Sun className="h-4 w-4 text-solar" />}
                title="Solar array upgrade"
                description="14–15 new panels"
                checked={active.upgrades.solar}
                onChange={(c) => updateActive((s) => (s.upgrades.solar = c))}
              />
            </div>

            <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
              {active.upgrades.hphws && (
                <ParamGroup title="HPHWS parameters" accent="hotwater">
                  <NumberParam
                    label="Heat pump COP"
                    value={Number((active.device_params.hphws_cop as number) ?? 3.5)}
                    min={2}
                    max={5}
                    step={0.1}
                    onChange={(v) => updateActive((s) => (s.device_params.hphws_cop = v))}
                  />
                </ParamGroup>
              )}
              {active.upgrades.induction && (
                <ParamGroup title="Induction parameters" accent="cooking">
                  <NumberParam
                    label="Efficiency ratio vs gas"
                    value={Number((active.device_params.induction_eff_ratio as number) ?? 3.0)}
                    min={1.5}
                    max={5}
                    step={0.1}
                    onChange={(v) => updateActive((s) => (s.device_params.induction_eff_ratio = v))}
                  />
                </ParamGroup>
              )}
              {active.upgrades.ev && (
                <ParamGroup title="EV parameters" accent="ev">
                  <NumberParam
                    label="kWh per 100 km"
                    value={Number((active.device_params.ev_kwh_per_100km as number) ?? 16)}
                    min={10}
                    max={25}
                    step={0.5}
                    onChange={(v) => updateActive((s) => (s.device_params.ev_kwh_per_100km = v))}
                  />
                  <div className="flex items-center justify-between rounded-md border p-3">
                    <div>
                      <Label>Include vehicle replacement cost</Label>
                      <p className="text-xs text-muted-foreground">Off = fuel-saving payback only</p>
                    </div>
                    <Switch
                      checked={Boolean(active.device_params.ev_include_vehicle)}
                      onCheckedChange={(c) => updateActive((s) => (s.device_params.ev_include_vehicle = c))}
                    />
                  </div>
                </ParamGroup>
              )}
              {active.upgrades.solar && (
                <ParamGroup title="Solar upgrade" accent="solar">
                  <div>
                    <Label className="mb-1.5 block">Panel option</Label>
                    <div className="flex gap-2">
                      {(["standard", "premium"] as const).map((p) => (
                        <Button
                          key={p}
                          variant={(active.device_params.solar_panel as string) === p ? "default" : "outline"}
                          size="sm"
                          className="flex-1"
                          onClick={() => updateActive((s) => (s.device_params.solar_panel = p))}
                        >
                          {p[0].toUpperCase() + p.slice(1)}
                        </Button>
                      ))}
                    </div>
                  </div>
                  <NumberParam
                    label="Panel count"
                    value={Number((active.device_params.solar_panel_count as number) ?? 15)}
                    min={10}
                    max={20}
                    step={1}
                    onChange={(v) => updateActive((s) => (s.device_params.solar_panel_count = v))}
                  />
                </ParamGroup>
              )}
            </div>

            {active.result?.breakdown && Object.keys(active.result.breakdown).length > 0 && (
              <Card className="bg-muted/30">
                <CardHeader>
                  <CardTitle className="text-base">Per-upgrade contribution</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
                    {Object.entries(active.result.breakdown).map(([k, v]: [string, any]) => (
                      <div key={k} className="rounded-md border bg-card p-3">
                        <div className="text-xs uppercase text-muted-foreground">{k}</div>
                        <div className="mt-1 text-lg font-semibold text-positive">{fmtAUD(v.annual_saving_aud)}/yr</div>
                        <div className="text-xs text-muted-foreground">{fmtKg(v.annual_co2_saving_kg)}/yr</div>
                        <div className="text-xs text-muted-foreground">capex {fmtAUD(v.capex_net_aud)}</div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </CardContent>
        </Card>
      )}

      <NewScenarioDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onCreate={async (s) => {
          await upsert(s);
          setDialogOpen(false);
        }}
      />
    </div>
  );
}

// ---------- supporting components ----------

const tagLabels: Record<string, string> = {
  best_payback: "Fastest payback",
  best_carbon: "Most carbon cut",
  best_20yr_value: "Best 20yr value",
  cheapest_entry: "Cheapest entry",
};

function UpgradeIcons({ upgrades }: { upgrades: Upgrades }) {
  const items: { on: boolean; el: React.ReactNode; key: string }[] = [
    { key: "hw", on: upgrades.hphws, el: <Flame className="h-3.5 w-3.5 text-hotwater" /> },
    { key: "in", on: upgrades.induction, el: <ChefHat className="h-3.5 w-3.5 text-cooking" /> },
    { key: "ev", on: upgrades.ev, el: <Car className="h-3.5 w-3.5 text-ev" /> },
    { key: "pv", on: upgrades.solar, el: <Sun className="h-3.5 w-3.5 text-solar" /> },
  ];
  const active = items.filter((i) => i.on);
  if (active.length === 0) return <span className="text-xs text-muted-foreground">do nothing</span>;
  return (
    <div className="flex items-center gap-1">
      {active.map((i) => (
        <div key={i.key} className="grid h-6 w-6 place-items-center rounded bg-muted">
          {i.el}
        </div>
      ))}
    </div>
  );
}

function RecommendationCards({ explore, onSave }: { explore: ExploreResult; onSave: (c: ExploreCombo) => void }) {
  const cards = [
    { idx: explore.best_payback_idx, icon: Zap, title: "Fastest payback", accent: "warning" as const, color: "text-solar" },
    { idx: explore.best_carbon_idx, icon: Leaf, title: "Most CO₂ cut", accent: "success" as const, color: "text-positive" },
    { idx: explore.best_npv_idx, icon: Award, title: "Best 20-yr value", accent: "default" as const, color: "text-primary" },
    { idx: explore.cheapest_idx, icon: DollarSign, title: "Cheapest entry", accent: "secondary" as const, color: "text-muted-foreground" },
  ];
  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
      {cards.map(({ idx, icon: Icon, title, color }) => {
        if (idx < 0) return null;
        const c = explore.combos[idx];
        return (
          <div key={title} className="rounded-lg border bg-card p-4 shadow-sm">
            <div className="mb-2 flex items-center justify-between">
              <div className={`flex items-center gap-2 text-xs font-medium uppercase tracking-wide ${color}`}>
                <Icon className="h-3.5 w-3.5" /> {title}
              </div>
            </div>
            <div className="text-base font-semibold">{c.label}</div>
            <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
              <span>{fmtAUD(c.capex_net_aud)} capex</span>
              <span className="text-positive">{fmtAUD(c.annual_saving_aud)}/yr</span>
              <span>{fmtYears(c.payback_years)}</span>
              <span>{fmtKg(c.annual_co2_saving_kg)}/yr</span>
            </div>
            <Button size="sm" variant="outline" className="mt-3 w-full" onClick={() => onSave(c)}>
              <Plus className="mr-1.5 h-3 w-3" /> Save as scenario
            </Button>
          </div>
        );
      })}
    </div>
  );
}

function NewScenarioDialog({
  open,
  onOpenChange,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreate: (s: Partial<Scenario>) => Promise<void>;
}) {
  const [name, setName] = useState("");
  const [desc, setDesc] = useState("");
  const [upgrades, setUpgrades] = useState<Upgrades>({ hphws: false, induction: false, ev: false, solar: false });
  const [busy, setBusy] = useState(false);

  const reset = () => {
    setName("");
    setDesc("");
    setUpgrades({ hphws: false, induction: false, ev: false, solar: false });
  };

  const submit = async () => {
    if (!name.trim()) {
      toast.error("Name required");
      return;
    }
    setBusy(true);
    try {
      await onCreate({
        name,
        description: desc,
        upgrades,
        device_params: {
          hphws_cop: 3.5,
          induction_eff_ratio: 3.0,
          ev_kwh_per_100km: 16,
          ev_include_vehicle: false,
          solar_panel: "premium",
          solar_panel_count: 15,
        },
        dispatch: {},
      } as Partial<Scenario>);
      reset();
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) reset();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New scenario</DialogTitle>
          <DialogDescription>
            Pick which upgrades to include. The scenario is computed immediately on create — you can fine-tune device parameters after.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          <div>
            <Label className="mb-1.5 block">Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Hot water + EV (smart charge)" autoFocus />
          </div>
          <div>
            <Label className="mb-1.5 block">Description (optional)</Label>
            <Input value={desc} onChange={(e) => setDesc(e.target.value)} placeholder="What you're testing here" />
          </div>
          <div>
            <Label className="mb-1.5 block">Upgrades</Label>
            <div className="grid grid-cols-2 gap-3">
              <ToggleCard
                icon={<Flame className="h-4 w-4 text-hotwater" />}
                title="Heat pump HW"
                description="Replace gas instant HW"
                checked={upgrades.hphws}
                onChange={(c) => setUpgrades((u) => ({ ...u, hphws: c }))}
              />
              <ToggleCard
                icon={<ChefHat className="h-4 w-4 text-cooking" />}
                title="Induction cooktop"
                description="Replace gas cooktop"
                checked={upgrades.induction}
                onChange={(c) => setUpgrades((u) => ({ ...u, induction: c }))}
              />
              <ToggleCard
                icon={<Car className="h-4 w-4 text-ev" />}
                title="Electric vehicle"
                description="Replace petrol CX5"
                checked={upgrades.ev}
                onChange={(c) => setUpgrades((u) => ({ ...u, ev: c }))}
              />
              <ToggleCard
                icon={<Sun className="h-4 w-4 text-solar" />}
                title="Solar upgrade"
                description="14–15 new panels"
                checked={upgrades.solar}
                onChange={(c) => setUpgrades((u) => ({ ...u, solar: c }))}
              />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={submit} disabled={busy || !name.trim()}>
            {busy ? "Creating…" : "Create & compute"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ToggleCard({ icon, title, description, checked, onChange }: { icon: React.ReactNode; title: string; description: string; checked: boolean; onChange: (c: boolean) => void }) {
  return (
    <div className={`flex items-center justify-between rounded-lg border p-4 transition-colors ${checked ? "border-primary bg-primary/5" : ""}`}>
      <div className="flex items-center gap-3">
        <div className="grid h-9 w-9 place-items-center rounded-md bg-muted">{icon}</div>
        <div>
          <div className="text-sm font-medium">{title}</div>
          <div className="text-xs text-muted-foreground">{description}</div>
        </div>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  );
}

function ParamGroup({ title, accent, children }: { title: string; accent: string; children: React.ReactNode }) {
  const accentBorder: Record<string, string> = {
    hotwater: "border-l-hotwater",
    cooking: "border-l-cooking",
    ev: "border-l-ev",
    solar: "border-l-solar",
  };
  return (
    <div className={`rounded-lg border border-l-4 bg-card p-4 ${accentBorder[accent] || "border-l-primary"}`}>
      <div className="mb-3 text-sm font-medium">{title}</div>
      <div className="space-y-3">{children}</div>
    </div>
  );
}

function NumberParam({ label, value, min, max, step, onChange }: { label: string; value: number; min: number; max: number; step: number; onChange: (v: number) => void }) {
  return (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <Label>{label}</Label>
        <span className="text-sm tabular-nums text-muted-foreground">{value}</span>
      </div>
      <Slider
        min={min}
        max={max}
        step={step}
        value={[value]}
        onValueChange={(v) => onChange(v[0])}
      />
    </div>
  );
}
