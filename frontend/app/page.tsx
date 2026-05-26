"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

import { useAuth } from "@/src/auth/AuthProvider";

export default function Home() {
  const { ready, session } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!ready) {
      return;
    }
    router.replace(session ? "/rooms" : "/login");
  }, [ready, router, session]);

  return <div className="p-6 text-zinc-300">Loading...</div>;
}
