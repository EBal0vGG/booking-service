"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

import { useAuth } from "@/src/auth/AuthProvider";
import { ReservationPrompt } from "@/src/components/ReservationPrompt";
import { ToastStack } from "@/src/components/ToastStack";
import type { UserRole } from "@/src/types/api";
import { useRealtime } from "@/src/ws/RealtimeProvider";

interface NavItem {
  href: string;
  label: string;
  roles: UserRole[];
}

const NAV_ITEMS: NavItem[] = [
  { href: "/rooms", label: "Rooms", roles: ["admin", "user"] },
  { href: "/bookings", label: "My bookings", roles: ["user"] },
  { href: "/reservations", label: "My reservations", roles: ["user"] },
  { href: "/admin/bookings", label: "Admin bookings", roles: ["admin"] },
];

function isAuthPage(pathname: string): boolean {
  return pathname.startsWith("/login") || pathname.startsWith("/register");
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { ready, session, logout } = useAuth();
  const { status } = useRealtime();

  const authPage = isAuthPage(pathname);

  useEffect(() => {
    if (!authPage && ready && !session) {
      router.replace("/login");
    }
  }, [authPage, ready, router, session]);

  if (!ready) {
    return <div className="p-6 text-zinc-300">Loading session...</div>;
  }

  if (!authPage && !session) {
    return <div className="p-6 text-zinc-300">Redirecting to login...</div>;
  }

  const onLogout = () => {
    logout();
    router.push("/login");
  };

  if (authPage) {
    return (
      <>
        <div className="mx-auto flex min-h-screen w-full max-w-xl flex-col justify-center px-4 py-10">
          {children}
        </div>
        <ToastStack />
        <ReservationPrompt />
      </>
    );
  }

  return (
    <>
      <div className="min-h-screen bg-zinc-950 text-zinc-100">
        <div className="mx-auto flex min-h-screen w-full max-w-7xl">
          <aside className="w-64 shrink-0 border-r border-zinc-800 p-4">
            <h1 className="text-lg font-semibold">Booking MVP</h1>
            <p className="mt-1 text-xs text-zinc-400">
              Role: {session?.role} | WS: {status}
            </p>
            <nav className="mt-4 flex flex-col gap-2">
              {NAV_ITEMS.filter((item) => session && item.roles.includes(session.role)).map(
                (item) => (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`rounded-md px-3 py-2 text-sm transition ${
                      pathname === item.href || pathname.startsWith(`${item.href}/`)
                        ? "bg-zinc-800 text-zinc-50"
                        : "text-zinc-300 hover:bg-zinc-900 hover:text-zinc-100"
                    }`}
                  >
                    {item.label}
                  </Link>
                ),
              )}
              {session?.role === "admin" ? (
                <Link
                  href="/rooms#new-room"
                  className="rounded-md px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-900 hover:text-zinc-100"
                >
                  Create room
                </Link>
              ) : null}
            </nav>
            <button
              type="button"
              className="mt-6 rounded-md border border-zinc-700 px-3 py-2 text-sm text-zinc-200 hover:border-zinc-500"
              onClick={onLogout}
            >
              Logout
            </button>
          </aside>
          <main className="flex-1 p-6">{children}</main>
        </div>
      </div>
      <ToastStack />
      <ReservationPrompt />
    </>
  );
}
