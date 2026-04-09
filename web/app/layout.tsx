import type { Metadata } from "next";
import "./globals.css";
import { Nav } from "@/components/nav";
import { Toaster } from "sonner";

export const metadata: Metadata = {
  title: "PHEM — Personalised Home Energy Model",
  description: "Evaluate the financial and carbon case for residential electrification",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark" suppressHydrationWarning>
      <body className="min-h-screen antialiased">
        <Nav />
        <main className="container py-8">{children}</main>
        <Toaster theme="dark" position="bottom-right" richColors closeButton />
      </body>
    </html>
  );
}
