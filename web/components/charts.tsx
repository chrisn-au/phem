"use client";

// Lightweight Recharts wrappers that pull colors from the Tailwind/CSS
// variable palette so dark mode and theme tweaks just work.

import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { ReactNode } from "react";

const palette: Record<string, string> = {
  solar:    "hsl(35 92% 55%)",
  grid:     "hsl(220 14% 50%)",
  battery:  "hsl(142 71% 45%)",
  gas:      "hsl(18 90% 55%)",
  ev:       "hsl(217 91% 60%)",
  hvac:     "hsl(199 89% 48%)",
  pool:     "hsl(188 78% 41%)",
  base:     "hsl(220 14% 65%)",
  cooking:  "hsl(340 82% 56%)",
  hotwater: "hsl(14 100% 57%)",
  positive: "hsl(142 71% 45%)",
  negative: "hsl(0 84% 60%)",
  primary:  "hsl(142 71% 36%)",
};

export const colors = palette;

interface FrameProps {
  children: ReactNode;
  height?: number;
}

export function ChartFrame({ children, height = 280 }: FrameProps) {
  return (
    <div className="w-full" style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        {children as any}
      </ResponsiveContainer>
    </div>
  );
}

interface SeriesDef {
  key: string;
  name: string;
  color: string;
  stack?: string;
}

interface DataChartProps<T extends Record<string, any>> {
  data: T[];
  xKey: keyof T & string;
  series: SeriesDef[];
  height?: number;
  xFormatter?: (v: any) => string;
  yFormatter?: (v: any) => string;
}

export function StackedAreaChart<T extends Record<string, any>>({
  data,
  xKey,
  series,
  height = 280,
  xFormatter,
  yFormatter,
}: DataChartProps<T>) {
  return (
    <ChartFrame height={height}>
      <AreaChart data={data} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
        <defs>
          {series.map((s) => (
            <linearGradient key={s.key} id={`g-${s.key}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={s.color} stopOpacity={0.8} />
              <stop offset="95%" stopColor={s.color} stopOpacity={0.05} />
            </linearGradient>
          ))}
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
        <XAxis dataKey={xKey} tickFormatter={xFormatter} stroke="hsl(var(--muted-foreground))" />
        <YAxis tickFormatter={yFormatter} stroke="hsl(var(--muted-foreground))" />
        <Tooltip
          contentStyle={{
            background: "hsl(var(--card))",
            border: "1px solid hsl(var(--border))",
            borderRadius: 8,
          }}
          formatter={(v: any, name: any) => [yFormatter ? yFormatter(v) : v, name]}
          labelFormatter={(v) => (xFormatter ? xFormatter(v) : String(v))}
        />
        <Legend wrapperStyle={{ fontSize: 12 }} />
        {series.map((s) => (
          <Area
            key={s.key}
            type="monotone"
            dataKey={s.key}
            name={s.name}
            stroke={s.color}
            strokeWidth={1.5}
            stackId={s.stack}
            fill={`url(#g-${s.key})`}
          />
        ))}
      </AreaChart>
    </ChartFrame>
  );
}

export function StackedBarChart<T extends Record<string, any>>({
  data,
  xKey,
  series,
  height = 280,
  xFormatter,
  yFormatter,
}: DataChartProps<T>) {
  return (
    <ChartFrame height={height}>
      <BarChart data={data} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
        <XAxis dataKey={xKey} tickFormatter={xFormatter} stroke="hsl(var(--muted-foreground))" />
        <YAxis tickFormatter={yFormatter} stroke="hsl(var(--muted-foreground))" />
        <Tooltip
          contentStyle={{
            background: "hsl(var(--card))",
            border: "1px solid hsl(var(--border))",
            borderRadius: 8,
          }}
          formatter={(v: any, name: any) => [yFormatter ? yFormatter(v) : v, name]}
          labelFormatter={(v) => (xFormatter ? xFormatter(v) : String(v))}
        />
        <Legend wrapperStyle={{ fontSize: 12 }} />
        {series.map((s) => (
          <Bar key={s.key} dataKey={s.key} name={s.name} stackId={s.stack || "stack"} fill={s.color} radius={[2, 2, 0, 0]} />
        ))}
      </BarChart>
    </ChartFrame>
  );
}

interface LineChartProps<T extends Record<string, any>> extends DataChartProps<T> {
  zeroLine?: boolean;
}

export function MultiLineChart<T extends Record<string, any>>({
  data,
  xKey,
  series,
  height = 280,
  xFormatter,
  yFormatter,
  zeroLine,
}: LineChartProps<T>) {
  return (
    <ChartFrame height={height}>
      <LineChart data={data} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
        <XAxis dataKey={xKey} tickFormatter={xFormatter} stroke="hsl(var(--muted-foreground))" />
        <YAxis tickFormatter={yFormatter} stroke="hsl(var(--muted-foreground))" />
        <Tooltip
          contentStyle={{
            background: "hsl(var(--card))",
            border: "1px solid hsl(var(--border))",
            borderRadius: 8,
          }}
          formatter={(v: any, name: any) => [yFormatter ? yFormatter(v) : v, name]}
          labelFormatter={(v) => (xFormatter ? xFormatter(v) : String(v))}
        />
        <Legend wrapperStyle={{ fontSize: 12 }} />
        {zeroLine && <ReferenceLine y={0} stroke="hsl(var(--muted-foreground))" strokeDasharray="4 4" />}
        {series.map((s) => (
          <Line
            key={s.key}
            type="monotone"
            dataKey={s.key}
            name={s.name}
            stroke={s.color}
            strokeWidth={2.5}
            dot={false}
            activeDot={{ r: 4 }}
          />
        ))}
      </LineChart>
    </ChartFrame>
  );
}
