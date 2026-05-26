export type UserRole = "admin" | "user";

export type ApiErrorCode =
  | "INVALID_REQUEST"
  | "UNAUTHORIZED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "ROOM_NOT_FOUND"
  | "SLOT_NOT_FOUND"
  | "BOOKING_NOT_FOUND"
  | "WAITLIST_NOT_FOUND"
  | "RESERVATION_NOT_FOUND"
  | "SLOT_ALREADY_BOOKED"
  | "SLOT_RESERVED"
  | "SLOT_NOT_BOOKED"
  | "SCHEDULE_EXISTS"
  | "WAITLIST_ALREADY_JOINED"
  | "INTERNAL_ERROR"
  | string;

export interface ErrorResponse {
  error: {
    code: ApiErrorCode;
    message: string;
  };
}

export interface Room {
  id: string;
  name: string;
  description?: string;
  capacity?: number;
}

export interface Slot {
  id: string;
  roomId: string;
  start: string;
  end: string;
}

export type SlotStatus = "available" | "booked" | "reserved" | "past";

export interface SlotView {
  id: string;
  roomId: string;
  start: string;
  end: string;
  status: SlotStatus;
  bookingId?: string;
  reservationId?: string;
  waitlistEntryId?: string;
}

export interface Booking {
  id: string;
  slotId: string;
  userId: string;
  status: string;
  conferenceLink?: string;
  createdAt: string;
}

export interface WaitlistEntry {
  id: string;
  slotId: string;
  userId: string;
  status: string;
  position: number;
  createdAt: string;
  notifiedAt?: string;
}

export interface SlotReservation {
  id: string;
  slotId: string;
  userId: string;
  waitlistEntryId?: string;
  status: string;
  expiresAt: string;
  createdAt: string;
  confirmedAt?: string;
  expiredAt?: string;
}

export interface Pagination {
  page: number;
  pageSize: number;
  total: number;
}

export type RealtimeMessageType =
  | "subscribed"
  | "unsubscribed"
  | "slot_booked"
  | "slot_released"
  | "slot_available"
  | "slot_reserved"
  | "slot_reservation_expired"
  | "waitlist_slot_reserved"
  | "waitlist_slot_available"
  | "reservation_expired"
  | "error";

export interface RealtimeMessage {
  type: RealtimeMessageType;
  roomId?: string;
  userId?: string;
  slotId?: string;
  bookingId?: string;
  reservationId?: string;
  waitlistEntryId?: string;
  expiresAt?: string;
  timestamp?: string;
  message?: string;
}

export interface AuthSession {
  token: string;
  role: UserRole;
  userId: string;
}
