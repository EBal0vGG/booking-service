"use client";

import { useEffect, useState } from "react";

import { listAdminBookings } from "@/src/api/bookings";
import { useRequireAuth } from "@/src/auth/useRequireAuth";
import {
  formatUtcDateTime,
  toUserErrorMessage,
} from "@/src/components/formatters";
import type { Booking, Pagination } from "@/src/types/api";

const PAGE_SIZE = 20;

export default function AdminBookingsPage() {
  const { ready, authorized } = useRequireAuth(["admin"]);

  const [bookings, setBookings] = useState<Booking[]>([]);
  const [pagination, setPagination] = useState<Pagination>({
    page: 1,
    pageSize: PAGE_SIZE,
    total: 0,
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadPage = async (page: number) => {
    setBusy(true);
    setError(null);
    try {
      const response = await listAdminBookings(page, PAGE_SIZE);
      setBookings(response.bookings);
      setPagination(response.pagination);
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    if (!ready || !authorized) {
      return;
    }
    const timeout = window.setTimeout(() => {
      void loadPage(1);
    }, 0);
    return () => window.clearTimeout(timeout);
  }, [authorized, ready]);

  if (!ready || !authorized) {
    return <div className="text-zinc-300">Loading...</div>;
  }

  const totalPages = Math.max(1, Math.ceil(pagination.total / pagination.pageSize));

  return (
    <div className="space-y-4">
      <header>
        <h2 className="text-2xl font-semibold">Admin bookings</h2>
        <p className="mt-1 text-sm text-zinc-400">GET /bookings/list?page=&pageSize=20</p>
      </header>

      <div className="flex items-center gap-2">
        <button
          type="button"
          className="rounded-md border border-zinc-700 px-3 py-2 text-sm hover:border-zinc-500 disabled:opacity-60"
          disabled={busy || pagination.page <= 1}
          onClick={() => void loadPage(pagination.page - 1)}
        >
          Prev
        </button>
        <button
          type="button"
          className="rounded-md border border-zinc-700 px-3 py-2 text-sm hover:border-zinc-500 disabled:opacity-60"
          disabled={busy || pagination.page >= totalPages}
          onClick={() => void loadPage(pagination.page + 1)}
        >
          Next
        </button>
        <span className="text-sm text-zinc-300">
          page {pagination.page}/{totalPages} (total {pagination.total})
        </span>
      </div>

      <div className="grid gap-3">
        {bookings.map((booking) => (
          <article
            key={booking.id}
            className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3"
          >
            <p className="text-sm">bookingId: {booking.id}</p>
            <p className="text-sm text-zinc-300">userId: {booking.userId}</p>
            <p className="text-sm text-zinc-300">slotId: {booking.slotId}</p>
            <p className="text-sm text-zinc-300">status: {booking.status}</p>
            <p className="text-sm text-zinc-300">
              createdAt: {formatUtcDateTime(booking.createdAt)}
            </p>
          </article>
        ))}
      </div>

      {!busy && bookings.length === 0 ? (
        <p className="text-sm text-zinc-400">No bookings found for this page.</p>
      ) : null}
      {error ? <p className="text-sm text-red-300">{error}</p> : null}
    </div>
  );
}
