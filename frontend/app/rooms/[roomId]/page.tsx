"use client";

import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";

import { createBooking } from "@/src/api/bookings";
import { createSchedule, listRoomSlots, listRooms } from "@/src/api/rooms";
import { joinWaitlist } from "@/src/api/waitlist";
import { useRequireAuth } from "@/src/auth/useRequireAuth";
import {
  formatUtcDateTime,
  toUserErrorMessage,
} from "@/src/components/formatters";
import type { Room, SlotStatus, SlotView } from "@/src/types/api";
import { useRealtime } from "@/src/ws/RealtimeProvider";

const ROOM_REFRESH_EVENTS = new Set([
  "slot_booked",
  "slot_released",
  "slot_reserved",
  "slot_available",
  "slot_reservation_expired",
]);

const WEEK_DAYS = [
  { id: 1, label: "Mon" },
  { id: 2, label: "Tue" },
  { id: 3, label: "Wed" },
  { id: 4, label: "Thu" },
  { id: 5, label: "Fri" },
  { id: 6, label: "Sat" },
  { id: 7, label: "Sun" },
];

function todayUTC(): string {
  return new Date().toISOString().slice(0, 10);
}

export default function RoomDetailsPage() {
  const params = useParams<{ roomId: string }>();
  const roomId = params.roomId;
  const { ready, session, authorized } = useRequireAuth();
  const { status, lastEvent, setRoomSubscription } = useRealtime();

  const [room, setRoom] = useState<Room | null>(null);
  const [slots, setSlots] = useState<SlotView[]>([]);
  const [date, setDate] = useState(todayUTC);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [scheduleDays, setScheduleDays] = useState<number[]>([1, 2, 3, 4, 5]);
  const [scheduleStart, setScheduleStart] = useState("09:00");
  const [scheduleEnd, setScheduleEnd] = useState("18:00");

  const canCreateSchedule = session?.role === "admin";
  const canBook = session?.role === "user";

  const roomTitle = useMemo(() => room?.name ?? `Room ${roomId}`, [room, roomId]);

  const loadRoom = useCallback(async () => {
    const response = await listRooms();
    setRoom(response.rooms.find((item) => item.id === roomId) ?? null);
  }, [roomId]);

  const loadSlots = useCallback(async () => {
    const response = await listRoomSlots(roomId, date);
    setSlots(response.slots);
  }, [date, roomId]);

  const refreshData = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      await Promise.all([loadRoom(), loadSlots()]);
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setBusy(false);
    }
  }, [loadRoom, loadSlots]);

  useEffect(() => {
    if (!ready || !authorized) {
      return;
    }
    const timeout = window.setTimeout(() => {
      void refreshData();
    }, 0);
    return () => window.clearTimeout(timeout);
  }, [authorized, ready, refreshData]);

  useEffect(() => {
    if (!roomId) {
      return;
    }
    setRoomSubscription(roomId);
    return () => setRoomSubscription(null);
  }, [roomId, setRoomSubscription]);

  useEffect(() => {
    if (!lastEvent?.roomId || lastEvent.roomId !== roomId) {
      return;
    }
    if (!ROOM_REFRESH_EVENTS.has(lastEvent.type)) {
      return;
    }
    const timeout = window.setTimeout(() => {
      void loadSlots();
    }, 0);
    return () => window.clearTimeout(timeout);
  }, [lastEvent, loadSlots, roomId]);

  const toggleDay = (value: number) => {
    setScheduleDays((current) =>
      current.includes(value)
        ? current.filter((item) => item !== value)
        : [...current, value].sort((a, b) => a - b),
    );
  };

  const handleCreateSchedule = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      await createSchedule(roomId, {
        daysOfWeek: scheduleDays,
        startTime: scheduleStart,
        endTime: scheduleEnd,
      });
      setMessage(
        "Расписание создано. Слоты появятся после фоновой генерации (обычно в течение минуты).",
      );
      await refreshData();
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const handleBook = async (slotId: string) => {
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      await createBooking(slotId, true);
      setMessage("Бронь успешно создана.");
      await loadSlots();
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const handleJoinWaitlist = async (slotId: string) => {
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const joined = await joinWaitlist(slotId);
      setMessage(
        `Добавлено в waitlist: entry=${joined.entry.id}, position=${joined.entry.position}.`,
      );
      await loadSlots();
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const groupedSlots = useMemo(() => {
    const map = new Map<string, SlotView[]>();
    for (const slot of slots) {
      const hour = new Date(slot.start).toISOString().slice(11, 13);
      if (!map.has(hour)) {
        map.set(hour, []);
      }
      map.get(hour)?.push(slot);
    }
    return Array.from(map.entries()).map(([hour, entries]) => ({
      hour,
      slots: entries.sort((a, b) => Date.parse(a.start) - Date.parse(b.start)),
    }));
  }, [slots]);

  const statusBadgeClass = (status: SlotStatus): string => {
    switch (status) {
      case "available":
        return "bg-emerald-900/50 text-emerald-300 border-emerald-700";
      case "booked":
        return "bg-rose-900/40 text-rose-300 border-rose-700";
      case "reserved":
        return "bg-amber-900/40 text-amber-300 border-amber-700";
      case "past":
        return "bg-zinc-800 text-zinc-300 border-zinc-600";
      default:
        return "bg-zinc-800 text-zinc-300 border-zinc-600";
    }
  };

  if (!ready || !authorized) {
    return <div className="text-zinc-300">Loading...</div>;
  }

  return (
    <div className="space-y-6">
      <header>
        <h2 className="text-2xl font-semibold">{roomTitle}</h2>
        <p className="mt-1 text-sm text-zinc-400">
          roomId: <span className="font-mono">{roomId}</span> | ws: {status}
        </p>
      </header>

      {canCreateSchedule ? (
        <section className="rounded-lg border border-zinc-800 bg-zinc-900 p-4">
          <h3 className="text-lg font-semibold">Create schedule (admin)</h3>
          <form className="mt-3 space-y-3" onSubmit={handleCreateSchedule}>
            <div>
              <p className="mb-1 text-sm text-zinc-300">Days of week</p>
              <div className="flex flex-wrap gap-2">
                {WEEK_DAYS.map((day) => (
                  <label key={day.id} className="flex items-center gap-2 text-sm text-zinc-200">
                    <input
                      type="checkbox"
                      checked={scheduleDays.includes(day.id)}
                      onChange={() => toggleDay(day.id)}
                    />
                    {day.label}
                  </label>
                ))}
              </div>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              <label className="text-sm">
                <span className="mb-1 block text-zinc-300">Start time (UTC)</span>
                <input
                  type="time"
                  className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2"
                  value={scheduleStart}
                  onChange={(event) => setScheduleStart(event.target.value)}
                  required
                />
              </label>
              <label className="text-sm">
                <span className="mb-1 block text-zinc-300">End time (UTC)</span>
                <input
                  type="time"
                  className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2"
                  value={scheduleEnd}
                  onChange={(event) => setScheduleEnd(event.target.value)}
                  required
                />
              </label>
            </div>
            <button
              type="submit"
              className="rounded-md bg-emerald-600 px-4 py-2 text-sm text-white hover:bg-emerald-500 disabled:opacity-60"
              disabled={busy}
            >
              Create schedule
            </button>
          </form>
        </section>
      ) : null}

      <section className="rounded-lg border border-zinc-800 bg-zinc-900 p-4">
        <h3 className="text-lg font-semibold">Room timeline (UTC)</h3>
        <label className="mt-2 block text-sm">
          <span className="mb-1 block text-zinc-300">Date (UTC)</span>
          <input
            type="date"
            className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2"
            value={date}
            onChange={(event) => setDate(event.target.value)}
          />
        </label>
        <button
          type="button"
          className="mt-2 rounded-md border border-zinc-700 px-3 py-2 text-sm hover:border-zinc-500"
          onClick={() => void loadSlots()}
        >
          Refresh slots
        </button>

        <div className="mt-4 space-y-4">
          {groupedSlots.map((group) => (
            <div key={group.hour}>
              <p className="mb-2 text-sm font-semibold text-zinc-300">{group.hour}:00</p>
              <div className="space-y-2">
                {group.slots.map((slot) => {
                  const startText = slot.start.slice(11, 16);
                  const endText = slot.end.slice(11, 16);
                  return (
                    <article
                      key={slot.id}
                      className="rounded-md border border-zinc-700 px-3 py-3"
                    >
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="w-32 text-sm text-zinc-100">
                          {startText} - {endText}
                        </span>
                        <span
                          className={`rounded border px-2 py-1 text-xs ${statusBadgeClass(slot.status)}`}
                        >
                          {slot.status}
                        </span>
                      </div>
                      <p className="mt-1 text-xs text-zinc-400">
                        slotId: <span className="font-mono">{slot.id}</span>
                      </p>
                      {slot.bookingId ? (
                        <p className="mt-1 text-xs text-zinc-400">
                          bookingId: <span className="font-mono">{slot.bookingId}</span>
                        </p>
                      ) : null}
                      {slot.reservationId ? (
                        <p className="mt-1 text-xs text-zinc-400">
                          reservationId:{" "}
                          <span className="font-mono">{slot.reservationId}</span>
                        </p>
                      ) : null}

                      {canBook && slot.status === "available" ? (
                        <button
                          type="button"
                          className="mt-3 rounded-md bg-emerald-600 px-3 py-2 text-sm text-white hover:bg-emerald-500 disabled:opacity-60"
                          onClick={() => void handleBook(slot.id)}
                          disabled={busy}
                        >
                          Book
                        </button>
                      ) : null}
                      {canBook && slot.status === "booked" ? (
                        <button
                          type="button"
                          className="mt-3 rounded-md bg-zinc-700 px-3 py-2 text-sm text-zinc-100 hover:bg-zinc-600 disabled:opacity-60"
                          onClick={() => void handleJoinWaitlist(slot.id)}
                          disabled={busy}
                        >
                          Join waitlist
                        </button>
                      ) : null}
                      {slot.status === "past" ? (
                        <p className="mt-2 text-xs text-zinc-500">
                          Past slot ({formatUtcDateTime(slot.start)})
                        </p>
                      ) : null}
                    </article>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
        {!busy && groupedSlots.length === 0 ? (
          <p className="mt-3 text-sm text-zinc-400">No slots for selected date.</p>
        ) : null}
      </section>

      {message ? <p className="text-sm text-emerald-300">{message}</p> : null}
      {error ? <p className="text-sm text-red-300">{error}</p> : null}
    </div>
  );
}
