"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Database, BarChart3, GitCompare, Settings, Zap } from "lucide-react";
import { cn } from "@/lib/utils";

const links = [
  { href: "/data",        label: "Data",        icon: Database },
  { href: "/baseline",    label: "Baseline",    icon: BarChart3 },
  { href: "/scenarios",   label: "Scenarios",   icon: GitCompare },
  { href: "/assumptions", label: "Assumptions", icon: Settings },
];

export function Nav() {
  const pathname = usePathname();
  return (
    <header className="sticky top-0 z-40 border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-16 items-center gap-6">
        <Link href="/" className="flex items-center gap-2">
          <div className="grid h-9 w-9 place-items-center rounded-lg bg-primary text-primary-foreground shadow-sm">
            <Zap className="h-5 w-5" strokeWidth={2.5} />
          </div>
          <div className="flex flex-col leading-tight">
            <span className="text-sm font-semibold tracking-tight">PHEM</span>
            <span className="text-[10px] uppercase text-muted-foreground tracking-wider">
              Personalised Home Energy Model
            </span>
          </div>
        </Link>
        <nav className="ml-6 flex items-center gap-1">
          {links.map(({ href, label, icon: Icon }) => {
            const active = pathname === href || (href !== "/" && pathname.startsWith(href));
            return (
              <Link
                key={href}
                href={href}
                className={cn(
                  "inline-flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  active
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-accent hover:text-foreground"
                )}
              >
                <Icon className="h-4 w-4" />
                {label}
              </Link>
            );
          })}
        </nav>
        <div className="ml-auto text-xs text-muted-foreground">NSW · 15-min · Phase 1</div>
      </div>
    </header>
  );
}
