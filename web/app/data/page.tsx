"use client";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { api, type DataHealth, type GasBill, uploadCSV } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Database, FileSpreadsheet, Flame, RefreshCw, Upload } from "lucide-react";

export default function DataPage() {
  const [health, setHealth] = useState<DataHealth | null>(null);
  const [bills, setBills] = useState<GasBill[]>([]);
  const [loading, setLoading] = useState(true);

  const [solarFile, setSolarFile] = useState<File | null>(null);
  const [consumptionFile, setConsumptionFile] = useState<File | null>(null);
  const [uploadingSolar, setUploadingSolar] = useState(false);
  const [uploadingCons, setUploadingCons] = useState(false);

  const [newBill, setNewBill] = useState<GasBill>({
    period_start: "",
    period_end: "",
    consumption_mj: 0,
    cost_aud: 0,
    note: "",
  });

  const refresh = async () => {
    try {
      const [h, b] = await Promise.all([api.dataHealth(), api.listGasBills()]);
      setHealth(h);
      setBills(b);
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const handleUpload = async (kind: "solar" | "consumption") => {
    const file = kind === "solar" ? solarFile : consumptionFile;
    if (!file) return;
    if (kind === "solar") setUploadingSolar(true);
    else setUploadingCons(true);
    try {
      const res = await uploadCSV(kind, file);
      toast.success(`${kind} uploaded — ${res.rows_loaded} rows (${res.gaps_found} gaps, ${res.duplicates} dup, ${res.implausible} odd)`);
      await refresh();
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      if (kind === "solar") setUploadingSolar(false);
      else setUploadingCons(false);
    }
  };

  const handleAddBill = async () => {
    try {
      await api.createGasBill(newBill);
      toast.success("Gas bill added");
      setNewBill({ period_start: "", period_end: "", consumption_mj: 0, cost_aud: 0, note: "" });
      await refresh();
    } catch (e: any) {
      toast.error(e.message);
    }
  };

  if (loading) return <div className="text-muted-foreground">Loading…</div>;

  return (
    <div className="space-y-8">
      <header className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Data</h1>
        <p className="text-muted-foreground">
          Upload Enphase solar production CSVs, Amber/Enphase consumption CSVs, and quarterly gas bills.
          On first boot the dashboard is pre-populated with synthetic data so you can see how it
          looks before loading real exports.
        </p>
      </header>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Stored intervals</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tabular-nums">{health?.total_intervals.toLocaleString("en-AU") || "0"}</div>
            <p className="mt-1 text-xs text-muted-foreground">15-minute energy_intervals rows</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Earliest reading</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{health?.range_start ? new Date(health.range_start).toLocaleDateString("en-AU") : "—"}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Latest reading</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{health?.range_end ? new Date(health.range_end).toLocaleDateString("en-AU") : "—"}</div>
          </CardContent>
        </Card>
      </section>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Upload className="h-5 w-5 text-solar" /> Solar production CSV
            </CardTitle>
            <CardDescription>Enphase Enlighten export, 15-min intervals. Existing rows are upserted.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Input type="file" accept=".csv" onChange={(e) => setSolarFile(e.target.files?.[0] || null)} />
            <Button onClick={() => handleUpload("solar")} disabled={!solarFile || uploadingSolar} className="w-full">
              {uploadingSolar ? "Uploading…" : "Upload solar CSV"}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Upload className="h-5 w-5 text-grid" /> Whole-home consumption CSV
            </CardTitle>
            <CardDescription>Amber Electric or Enphase consumption export, 15-min intervals.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Input type="file" accept=".csv" onChange={(e) => setConsumptionFile(e.target.files?.[0] || null)} />
            <Button onClick={() => handleUpload("consumption")} disabled={!consumptionFile || uploadingCons} className="w-full">
              {uploadingCons ? "Uploading…" : "Upload consumption CSV"}
            </Button>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Flame className="h-5 w-5 text-gas" /> Quarterly gas bills
          </CardTitle>
          <CardDescription>Enter bill totals manually — the model splits 80/20 hot water/cooking by default.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-5">
            <div>
              <Label className="mb-1.5 block">From</Label>
              <Input type="date" value={newBill.period_start} onChange={(e) => setNewBill({ ...newBill, period_start: e.target.value })} />
            </div>
            <div>
              <Label className="mb-1.5 block">To</Label>
              <Input type="date" value={newBill.period_end} onChange={(e) => setNewBill({ ...newBill, period_end: e.target.value })} />
            </div>
            <div>
              <Label className="mb-1.5 block">Consumption (MJ)</Label>
              <Input
                type="number"
                value={newBill.consumption_mj || ""}
                onChange={(e) => setNewBill({ ...newBill, consumption_mj: parseFloat(e.target.value || "0") })}
              />
            </div>
            <div>
              <Label className="mb-1.5 block">Cost (AUD)</Label>
              <Input
                type="number"
                value={newBill.cost_aud || ""}
                onChange={(e) => setNewBill({ ...newBill, cost_aud: parseFloat(e.target.value || "0") })}
              />
            </div>
            <div className="flex items-end">
              <Button onClick={handleAddBill} className="w-full">Add bill</Button>
            </div>
          </div>

          <div className="overflow-x-auto rounded-md border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr className="text-left">
                  <th className="px-3 py-2 font-medium">Period</th>
                  <th className="px-3 py-2 font-medium text-right">Consumption (MJ)</th>
                  <th className="px-3 py-2 font-medium text-right">Cost</th>
                  <th className="px-3 py-2 font-medium">Note</th>
                </tr>
              </thead>
              <tbody>
                {bills.length === 0 && (
                  <tr><td colSpan={4} className="px-3 py-6 text-center text-muted-foreground">No gas bills yet.</td></tr>
                )}
                {bills.map((b) => (
                  <tr key={b.id} className="border-t">
                    <td className="px-3 py-2">
                      {new Date(b.period_start).toLocaleDateString("en-AU")} – {new Date(b.period_end).toLocaleDateString("en-AU")}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">{Math.round(b.consumption_mj).toLocaleString("en-AU")}</td>
                    <td className="px-3 py-2 text-right tabular-nums">${Math.round(b.cost_aud || 0)}</td>
                    <td className="px-3 py-2 text-muted-foreground">{b.note || ""}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <FileSpreadsheet className="h-5 w-5" /> Ingestion history
            </CardTitle>
            <CardDescription>Audit log of every dataset loaded into PHEM</CardDescription>
          </div>
          <Button variant="outline" size="sm" onClick={refresh}>
            <RefreshCw className="mr-2 h-4 w-4" /> Refresh
          </Button>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto rounded-md border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr className="text-left">
                  <th className="px-3 py-2 font-medium">Source</th>
                  <th className="px-3 py-2 font-medium">File</th>
                  <th className="px-3 py-2 font-medium text-right">Rows</th>
                  <th className="px-3 py-2 font-medium text-right">Gaps</th>
                  <th className="px-3 py-2 font-medium text-right">Dup</th>
                  <th className="px-3 py-2 font-medium text-right">Odd</th>
                  <th className="px-3 py-2 font-medium">Range</th>
                  <th className="px-3 py-2 font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {(health?.runs || []).map((r) => (
                  <tr key={r.id} className="border-t">
                    <td className="px-3 py-2">{r.source}</td>
                    <td className="px-3 py-2 text-muted-foreground">{r.filename || "—"}</td>
                    <td className="px-3 py-2 text-right tabular-nums">{r.rows_loaded.toLocaleString("en-AU")}</td>
                    <td className="px-3 py-2 text-right tabular-nums">{r.gaps_found}</td>
                    <td className="px-3 py-2 text-right tabular-nums">{r.duplicates}</td>
                    <td className="px-3 py-2 text-right tabular-nums">{r.implausible}</td>
                    <td className="px-3 py-2 text-xs text-muted-foreground">
                      {r.range_start ? new Date(r.range_start).toLocaleDateString("en-AU") : ""}
                      {r.range_end ? ` → ${new Date(r.range_end).toLocaleDateString("en-AU")}` : ""}
                    </td>
                    <td className="px-3 py-2">
                      <Badge variant={r.status === "ok" ? "success" : r.status === "partial" ? "warning" : "danger"}>{r.status}</Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
