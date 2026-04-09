import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

interface StatCardProps {
  label: string;
  value: string;
  hint?: string;
  icon?: LucideIcon;
  accent?: "solar" | "grid" | "battery" | "gas" | "ev" | "hvac" | "positive" | "negative" | "primary";
}

const accentClasses: Record<NonNullable<StatCardProps["accent"]>, string> = {
  solar: "text-solar bg-solar/10",
  grid: "text-grid bg-grid/10",
  battery: "text-battery bg-battery/10",
  gas: "text-gas bg-gas/10",
  ev: "text-ev bg-ev/10",
  hvac: "text-hvac bg-hvac/10",
  positive: "text-positive bg-positive/10",
  negative: "text-negative bg-negative/10",
  primary: "text-primary bg-primary/10",
};

export function StatCard({ label, value, hint, icon: Icon, accent = "primary" }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
        {Icon && (
          <div className={cn("grid h-8 w-8 place-items-center rounded-md", accentClasses[accent])}>
            <Icon className="h-4 w-4" />
          </div>
        )}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold tabular-nums tracking-tight">{value}</div>
        {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
      </CardContent>
    </Card>
  );
}
