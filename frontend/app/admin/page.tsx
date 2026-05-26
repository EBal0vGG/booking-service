"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function AdminPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/admin/bookings");
  }, [router]);

  return <div className="text-zinc-300">Redirecting...</div>;
}
