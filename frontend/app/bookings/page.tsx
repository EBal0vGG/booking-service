"use client";

import { useEffect, useState } from "react";

import { cancelBooking, listMyBookings } from "@/src/api/bookings";
import { useRequireAuth } from "@/src/auth/useRequireAuth";
import {
  formatUtcDateTime,
  toUserErrorMessage,
} from "@/src/components/formatters";
import type { Booking } from "@/src/types/api";

export default function MyBookingsPage() {
  const { ready, authorized } = useRequireAuth(["user"]);
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadBookings = async () => {
    setBusy(true);
    setError(null);
    try {
      const response = await listMyBookings();
      setBookings(response.bookings);
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
      void loadBookings();
    }, 0);
    return () => window.clearTimeout(timeout);
  }, [authorized, ready]);

  const handleCancel = async (bookingId: string) => {
    setBusy(true);
    setError(null);
    try {
      await cancelBooking(bookingId);
      await loadBookings();
    } catch (err) {
      setError(toUserErrorMessage(err));
      setBusy(false);
    }
  };

  if (!ready || !authorized) {
    return <div className="text-zinc-300">Loading...</div>;
  }

  return (
    <div className="space-y-4">
      <header>
        <h2 className="text-2xl font-semibold">My bookings</h2>
        <p className="mt-1 text-sm text-zinc-400">GET /bookings/my</p>
      </header>

      <button
        type="button"
        className="rounded-md border border-zinc-700 px-3 py-2 text-sm hover:border-zinc-500"
        onClick={() => void loadBookings()}
      >
        Refresh
      </button>

      <div className="grid gap-3">
        {bookings.map((booking) => (
          <article
            key={booking.id}
            className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3"
          >
            <p className="text-sm">bookingId: {booking.id}</p>
            <p className="text-sm text-zinc-300">slotId: {booking.slotId}</p>
            <p className="text-sm text-zinc-300">status: {booking.status}</p>
            <p className="text-sm text-zinc-300">
              createdAt: {formatUtcDateTime(booking.createdAt)}
            </p>
            <button
              type="button"
              className="mt-3 rounded-md bg-zinc-700 px-3 py-2 text-sm hover:bg-zinc-600 disabled:opacity-60"
              onClick={() => void handleCancel(booking.id)}
              disabled={busy}
            >
              Cancel
            </button>
          </article>
        ))}
      </div>

      {!busy && bookings.length === 0 ? (
        <p className="text-sm text-zinc-400">No bookings yet.</p>
      ) : null}
      {error ? <p className="text-sm text-red-300">{error}</p> : null}
    </div>
  );
}
