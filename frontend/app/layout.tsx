import type { Metadata } from "next";
import "./globals.css";
import { AppShell } from "@/src/components/AppShell";
import { Providers } from "@/src/components/Providers";

export const metadata: Metadata = {
  title: "Booking Frontend MVP",
  description: "Manual testing UI for booking backend flow",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased">
      <body className="min-h-full bg-zinc-950 text-zinc-100">
        <Providers>
          <AppShell>{children}</AppShell>
        </Providers>
      </body>
    </html>
  );
}
