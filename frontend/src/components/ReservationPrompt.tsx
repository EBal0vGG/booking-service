"use client";

import { useEffect, useState } from "react";

import { useRealtime } from "@/src/ws/RealtimeProvider";

function formatCountdown(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

export function ReservationPrompt() {
  const {
    activeReservation,
    confirmActiveReservation,
    cancelActiveReservation,
    dismissReservationPrompt,
  } = useRealtime();
  const [busy, setBusy] = useState(false);
  const [nowMs, setNowMs] = useState(() => Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => {
      setNowMs(Date.now());
    }, 1000);
    return () => window.clearInterval(timer);
  }, []);

  if (!activeReservation) {
    return null;
  }

  const expiresAtMs = Date.parse(activeReservation.expiresAt);
  const remainingSeconds = Number.isNaN(expiresAtMs)
    ? 0
    : Math.max(0, Math.floor((expiresAtMs - nowMs) / 1000));
  const expired = remainingSeconds <= 0;

  const handleConfirm = async () => {
    setBusy(true);
    await confirmActiveReservation();
    setBusy(false);
  };

  const handleCancel = async () => {
    setBusy(true);
    await cancelActiveReservation();
    setBusy(false);
  };

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/70 px-4">
      <div className="w-full max-w-lg rounded-xl border border-zinc-700 bg-zinc-900 p-5">
        <h2 className="text-lg font-semibold text-zinc-100">Временный резерв слота</h2>
        <p className="mt-2 text-sm text-zinc-300">
          Слот <span className="font-mono">{activeReservation.slotId}</span> доступен для
          подтверждения.
        </p>
        <p className="mt-2 text-sm text-zinc-300">
          Подтвердите до{" "}
          <span className="font-semibold">{new Date(activeReservation.expiresAt).toUTCString()}</span>.
        </p>
        <p className="mt-3 text-2xl font-bold text-emerald-400">
          {formatCountdown(Math.max(remainingSeconds, 0))}
        </p>

        {expired ? (
          <p className="mt-2 text-sm text-red-300">Резерв истёк. Обновите слоты.</p>
        ) : null}
        <p className="mt-2 text-xs text-zinc-400">
          Можно открыть позже в My reservations.
        </p>

        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            className="rounded-md bg-emerald-600 px-3 py-2 text-sm text-white hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-60"
            onClick={handleConfirm}
            disabled={busy || expired}
          >
            Confirm reservation
          </button>
          <button
            type="button"
            className="rounded-md bg-zinc-700 px-3 py-2 text-sm text-zinc-100 hover:bg-zinc-600 disabled:cursor-not-allowed disabled:opacity-60"
            onClick={handleCancel}
            disabled={busy || expired}
          >
            Cancel reservation
          </button>
          <button
            type="button"
            className="rounded-md border border-zinc-600 px-3 py-2 text-sm text-zinc-300 hover:border-zinc-400 hover:text-zinc-100"
            onClick={dismissReservationPrompt}
          >
            View later
          </button>
        </div>
      </div>
    </div>
  );
}
