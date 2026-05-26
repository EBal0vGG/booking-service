"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";

import { useAuth } from "@/src/auth/AuthProvider";
import type { UserRole } from "@/src/types/api";

export function useRequireAuth(allowedRoles?: UserRole[]) {
  const { ready, session } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!ready) {
      return;
    }
    if (!session) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
      return;
    }
    if (allowedRoles && !allowedRoles.includes(session.role)) {
      router.replace("/rooms");
    }
  }, [allowedRoles, pathname, ready, router, session]);

  const authorized =
    !!session && (!allowedRoles || allowedRoles.includes(session.role));

  return {
    ready,
    session,
    authorized,
    loading: !ready || !authorized,
  };
}
