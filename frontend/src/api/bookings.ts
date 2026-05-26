import { apiFetch } from "@/src/api/client";
import type { Booking, Pagination } from "@/src/types/api";

interface BookingResponse {
  booking: Booking;
}

interface MyBookingsResponse {
  bookings: Booking[];
}

interface AdminBookingsResponse {
  bookings: Booking[];
  pagination: Pagination;
}

export function createBooking(
  slotId: string,
  createConferenceLink = false,
): Promise<BookingResponse> {
  return apiFetch<BookingResponse>("/bookings/create", {
    method: "POST",
    body: JSON.stringify({ slotId, createConferenceLink }),
  });
}

export function cancelBooking(bookingId: string): Promise<BookingResponse> {
  return apiFetch<BookingResponse>(`/bookings/${bookingId}/cancel`, {
    method: "POST",
  });
}

export function listMyBookings(): Promise<MyBookingsResponse> {
  return apiFetch<MyBookingsResponse>("/bookings/my");
}

export function listAdminBookings(
  page: number,
  pageSize: number,
): Promise<AdminBookingsResponse> {
  const query = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  });
  return apiFetch<AdminBookingsResponse>(`/bookings/list?${query.toString()}`);
}
