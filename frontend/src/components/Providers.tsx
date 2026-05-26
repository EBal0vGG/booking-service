"use client";

import { AuthProvider } from "@/src/auth/AuthProvider";
import { RealtimeProvider } from "@/src/ws/RealtimeProvider";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <RealtimeProvider>{children}</RealtimeProvider>
    </AuthProvider>
  );
}
