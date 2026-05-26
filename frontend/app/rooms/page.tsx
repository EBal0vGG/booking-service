"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";

import { createRoom, listRooms } from "@/src/api/rooms";
import { useRequireAuth } from "@/src/auth/useRequireAuth";
import { toUserErrorMessage } from "@/src/components/formatters";
import type { Room } from "@/src/types/api";

export default function RoomsPage() {
  const { ready, session, authorized } = useRequireAuth();
  const router = useRouter();

  const [rooms, setRooms] = useState<Room[]>([]);
  const [loadingRooms, setLoadingRooms] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [capacity, setCapacity] = useState("4");
  const [creating, setCreating] = useState(false);

  const canCreateRoom = session?.role === "admin";

  const loadRooms = async () => {
    setLoadingRooms(true);
    setError(null);
    try {
      const response = await listRooms();
      setRooms(response.rooms);
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setLoadingRooms(false);
    }
  };

  useEffect(() => {
    if (!ready || !authorized) {
      return;
    }
    const timeout = window.setTimeout(() => {
      void loadRooms();
    }, 0);
    return () => window.clearTimeout(timeout);
  }, [authorized, ready]);

  const handleCreateRoom = async (event: FormEvent) => {
    event.preventDefault();
    setCreating(true);
    setError(null);
    try {
      const created = await createRoom({
        name,
        description: description.trim() || undefined,
        capacity: capacity.trim() ? Number(capacity) : undefined,
      });
      await loadRooms();
      router.push(`/rooms/${created.room.id}`);
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setCreating(false);
    }
  };

  if (!ready || !authorized) {
    return <div className="text-zinc-300">Loading...</div>;
  }

  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-2xl font-semibold">Rooms</h2>
        <p className="mt-1 text-sm text-zinc-400">
          Список переговорок. Время в UI показывается в UTC.
        </p>
      </section>

      <section className="rounded-lg border border-zinc-800">
        <div className="border-b border-zinc-800 px-4 py-3 text-sm text-zinc-300">
          Available rooms
        </div>
        <div className="p-4">
          {loadingRooms ? <p className="text-sm text-zinc-400">Loading rooms...</p> : null}
          {!loadingRooms && rooms.length === 0 ? (
            <p className="text-sm text-zinc-400">No rooms yet.</p>
          ) : null}
          <div className="grid gap-3">
            {rooms.map((room) => (
              <article
                key={room.id}
                className="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-3"
              >
                <h3 className="text-lg font-medium">{room.name}</h3>
                <p className="mt-1 text-sm text-zinc-400">
                  id: <span className="font-mono">{room.id}</span>
                </p>
                {room.description ? (
                  <p className="mt-1 text-sm text-zinc-300">{room.description}</p>
                ) : null}
                {room.capacity ? (
                  <p className="mt-1 text-sm text-zinc-300">capacity: {room.capacity}</p>
                ) : null}
                <Link
                  href={`/rooms/${room.id}`}
                  className="mt-3 inline-block rounded-md bg-zinc-700 px-3 py-2 text-sm hover:bg-zinc-600"
                >
                  Open room
                </Link>
              </article>
            ))}
          </div>
        </div>
      </section>

      {canCreateRoom ? (
        <section id="new-room" className="rounded-lg border border-zinc-800 bg-zinc-900 p-4">
          <h3 className="text-lg font-semibold">Create room (admin)</h3>
          <form className="mt-3 grid gap-3 md:grid-cols-2" onSubmit={handleCreateRoom}>
            <label className="text-sm">
              <span className="mb-1 block text-zinc-300">Name</span>
              <input
                className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2"
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
              />
            </label>
            <label className="text-sm">
              <span className="mb-1 block text-zinc-300">Capacity</span>
              <input
                type="number"
                min={1}
                className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2"
                value={capacity}
                onChange={(event) => setCapacity(event.target.value)}
              />
            </label>
            <label className="text-sm md:col-span-2">
              <span className="mb-1 block text-zinc-300">Description</span>
              <textarea
                className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
              />
            </label>
            <div className="md:col-span-2">
              <button
                type="submit"
                className="rounded-md bg-emerald-600 px-4 py-2 text-sm text-white hover:bg-emerald-500 disabled:opacity-60"
                disabled={creating}
              >
                Create room
              </button>
            </div>
          </form>
        </section>
      ) : null}

      {error ? <p className="text-sm text-red-300">{error}</p> : null}
    </div>
  );
}
