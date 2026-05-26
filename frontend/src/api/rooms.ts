import { apiFetch } from "@/src/api/client";
import type { Room, Slot, SlotView } from "@/src/types/api";

interface ListRoomsResponse {
  rooms: Room[];
}

interface CreateRoomResponse {
  room: Room;
}

interface CreateScheduleResponse {
  schedule: {
    roomId: string;
    daysOfWeek: number[];
    startTime: string;
    endTime: string;
  };
}

interface ListSlotsResponse {
  slots: Slot[];
}

interface ListSlotViewsResponse {
  slots: SlotView[];
}

export function listRooms(): Promise<ListRoomsResponse> {
  return apiFetch<ListRoomsResponse>("/rooms/list");
}

export function createRoom(input: {
  name: string;
  description?: string;
  capacity?: number;
}): Promise<CreateRoomResponse> {
  return apiFetch<CreateRoomResponse>("/rooms/create", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function createSchedule(
  roomId: string,
  input: { daysOfWeek: number[]; startTime: string; endTime: string },
): Promise<CreateScheduleResponse> {
  return apiFetch<CreateScheduleResponse>(`/rooms/${roomId}/schedule/create`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function listAvailableSlots(
  roomId: string,
  date: string,
): Promise<ListSlotsResponse> {
  const query = new URLSearchParams({ date });
  return apiFetch<ListSlotsResponse>(`/rooms/${roomId}/slots/list?${query.toString()}`);
}

export function listRoomSlots(roomId: string, date: string): Promise<ListSlotViewsResponse> {
  const query = new URLSearchParams({ date });
  return apiFetch<ListSlotViewsResponse>(`/rooms/${roomId}/slots/all?${query.toString()}`);
}
