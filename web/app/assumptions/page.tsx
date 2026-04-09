"use client";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { api, type Assumption } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Save, Settings } from "lucide-react";

const categoryOrder = ["site", "usage", "cost", "rebate", "panel", "dispatch", "emission"];
const categoryLabel: Record<string, string> = {
  site: "Site",
  usage: "Usage",
  cost: "Costs",
  rebate: "Rebates",
  panel: "Panel specs",
  dispatch: "Dispatch heuristics",
  emission: "Emissions factors",
};

export default function AssumptionsPage() {
  const [assumptions, setAssumptions] = useState<Assumption[]>([]);
  const [edits, setEdits] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.listAssumptions()
      .then((list) => {
        setAssumptions(list);
        setLoading(false);
      })
      .catch((e) => toast.error(e.message));
  }, []);

  const grouped = assumptions.reduce<Record<string, Assumption[]>>((acc, a) => {
    (acc[a.category] ||= []).push(a);
    return acc;
  }, {});

  const handleSave = async (a: Assumption) => {
    const raw = edits[a.key];
    if (raw == null) return;
    let parsed: unknown;
    if (typeof a.value === "object" && a.value !== null) {
      try {
        parsed = JSON.parse(raw);
      } catch {
        toast.error("Invalid JSON");
        return;
      }
    } else if (typeof a.value === "number") {
      parsed = parseFloat(raw);
      if (Number.isNaN(parsed as number)) {
        toast.error("Not a number");
        return;
      }
    } else if (typeof a.value === "boolean") {
      parsed = raw === "true";
    } else {
      parsed = raw;
    }
    setSaving(true);
    try {
      await api.updateAssumption(a.key, parsed);
      const next = assumptions.map((x) => (x.key === a.key ? { ...x, value: parsed } : x));
      setAssumptions(next);
      setEdits((e) => {
        const c = { ...e };
        delete c[a.key];
        return c;
      });
      toast.success(`Saved ${a.label}`);
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="text-muted-foreground">Loading…</div>;

  return (
    <div className="space-y-8">
      <header className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Assumptions</h1>
        <p className="text-muted-foreground">
          Every model assumption is editable. Saving any value automatically recomputes all
          scenarios so the dashboard reflects your local pricing and rebate landscape (NFR-06).
        </p>
      </header>

      {categoryOrder.filter((c) => grouped[c]).map((category) => (
        <Card key={category}>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5 text-muted-foreground" /> {categoryLabel[category] || category}
            </CardTitle>
            <CardDescription>{grouped[category].length} parameters</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {grouped[category].map((a) => {
                const isObject = typeof a.value === "object" && a.value !== null;
                const initial = isObject ? JSON.stringify(a.value, null, 2) : String(a.value ?? "");
                const current = edits[a.key] ?? initial;
                const dirty = edits[a.key] !== undefined && edits[a.key] !== initial;
                return (
                  <div key={a.key} className="rounded-md border bg-card p-4">
                    <div className="mb-2 flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <Label className="block">{a.label}</Label>
                        {a.description && <p className="text-xs text-muted-foreground">{a.description}</p>}
                        <p className="mt-1 font-mono text-[10px] text-muted-foreground/60">{a.key}</p>
                      </div>
                      {a.unit && <Badge variant="outline">{a.unit}</Badge>}
                    </div>
                    {isObject ? (
                      <textarea
                        className="font-mono mt-2 h-32 w-full rounded-md border border-input bg-background px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-ring"
                        value={current}
                        onChange={(e) => setEdits((s) => ({ ...s, [a.key]: e.target.value }))}
                      />
                    ) : (
                      <Input
                        className="mt-2"
                        value={current}
                        onChange={(e) => setEdits((s) => ({ ...s, [a.key]: e.target.value }))}
                        type={typeof a.value === "number" ? "number" : "text"}
                        step="any"
                      />
                    )}
                    <div className="mt-3 flex justify-end">
                      <Button size="sm" variant={dirty ? "default" : "outline"} disabled={!dirty || saving} onClick={() => handleSave(a)}>
                        <Save className="mr-2 h-3 w-3" /> Save
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
