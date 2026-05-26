"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import {
  cancelReservation,
  confirmReservation,
  listMyActiveReservations,
} from "@/src/api/reservations";
import { useRequireAuth } from "@/src/auth/useRequireAuth";
import { formatUtcDateTime, toUserErrorMessage } from "@/src/components/formatters";
import type { SlotReservation } from "@/src/types/api";
import { useRealtime } from "@/src/ws/RealtimeProvider";

function formatCountdown(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

function remainingSeconds(expiresAt: string, nowMs: number): number {
  const parsed = Date.parse(expiresAt);
  if (Number.isNaN(parsed)) {
    return 0;
  }
  return Math.max(0, Math.floor((parsed - nowMs) / 1000));
}

export default function MyReservationsPage() {
  const { ready, authorized } = useRequireAuth(["user"]);
  const { reloadActiveReservations } = useRealtime();
  const router = useRouter();

  const [reservations, setReservations] = useState<SlotReservation[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [activeAction, setActiveAction] = useState<string | null>(null);
  const [nowMs, setNowMs] = useState(() => Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => {
      setNowMs(Date.now());
    }, 1000);
    return () => window.clearInterval(timer);
  }, []);

  const loadReservations = async () => {
    setBusy(true);
    setError(null);
    try {
      const response = await listMyActiveReservations();
      setReservations(response.reservations);
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
      void loadReservations();
    }, 0);
    return () => window.clearTimeout(timeout);
  }, [authorized, ready]);

  const sortedReservations = useMemo(
    () =>
      [...reservations].sort(
        (a, b) => Date.parse(a.expiresAt) - Date.parse(b.expiresAt),
      ),
    [reservations],
  );

  const handleConfirm = async (reservationId: string) => {
    setActiveAction(`confirm:${reservationId}`);
    setError(null);
    setMessage(null);
    try {
      await confirmReservation(reservationId);
      await Promise.all([loadReservations(), reloadActiveReservations()]);
      setMessage("Резерв подтверждён, открываю мои брони.");
      router.push("/bookings");
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setActiveAction(null);
    }
  };

  const handleCancel = async (reservationId: string) => {
    setActiveAction(`cancel:${reservationId}`);
    setError(null);
    setMessage(null);
    try {
      await cancelReservation(reservationId);
      await Promise.all([loadReservations(), reloadActiveReservations()]);
      setMessage("Резерв отменён.");
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setActiveAction(null);
    }
  };

  if (!ready || !authorized) {
    return <div className="text-zinc-300">Loading...</div>;
  }

  return (
    <div className="space-y-4">
      <header>
        <h2 className="text-2xl font-semibold">My reservations</h2>
        <p className="mt-1 text-sm text-zinc-400">GET /reservations/my/active</p>
      </header>

      <button
        type="button"
        className="rounded-md border border-zinc-700 px-3 py-2 text-sm hover:border-zinc-500"
        onClick={() => void loadReservations()}
      >
        Refresh
      </button>

      <div className="grid gap-3">
        {sortedReservations.map((reservation) => {
          const countdown = remainingSeconds(reservation.expiresAt, nowMs);
          const expired = countdown <= 0;
          return (
            <article
              key={reservation.id}
              className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3"
            >
              <p className="text-sm">reservationId: {reservation.id}</p>
              <p className="text-sm text-zinc-300">slotId: {reservation.slotId}</p>
              <p className="text-sm text-zinc-300">
                expiresAt: {formatUtcDateTime(reservation.expiresAt)}
              </p>
              <p className="mt-1 text-lg font-semibold text-emerald-400">
                {formatCountdown(countdown)}
              </p>
              {expired ? <p className="text-sm text-red-300">Reservation expired.</p> : null}

              <div className="mt-3 flex flex-wrap gap-2">
                <button
                  type="button"
                  className="rounded-md bg-emerald-600 px-3 py-2 text-sm text-white hover:bg-emerald-500 disabled:opacity-60"
                  onClick={() => void handleConfirm(reservation.id)}
                  disabled={activeAction !== null || expired}
                >
                  {activeAction === `confirm:${reservation.id}` ? "Confirming..." : "Confirm"}
                </button>
                <button
                  type="button"
                  className="rounded-md bg-zinc-700 px-3 py-2 text-sm hover:bg-zinc-600 disabled:opacity-60"
                  onClick={() => void handleCancel(reservation.id)}
                  disabled={activeAction !== null || expired}
                >
                  {activeAction === `cancel:${reservation.id}` ? "Cancelling..." : "Cancel"}
                </button>
              </div>
            </article>
          );
        })}
      </div>

      {!busy && sortedReservations.length === 0 ? (
        <p className="text-sm text-zinc-400">No active reservations.</p>
      ) : null}
      {message ? <p className="text-sm text-emerald-300">{message}</p> : null}
      {error ? <p className="text-sm text-red-300">{error}</p> : null}
    </div>
  );
}
