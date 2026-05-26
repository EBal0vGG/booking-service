import { apiFetch } from "@/src/api/client";
import type { Booking, SlotReservation } from "@/src/types/api";

interface ConfirmReservationResponse {
  booking: Booking;
  reservation: SlotReservation;
}

interface CancelReservationResponse {
  reservation: SlotReservation;
}

interface ListMyActiveReservationsResponse {
  reservations: SlotReservation[];
}

export function confirmReservation(
  reservationId: string,
): Promise<ConfirmReservationResponse> {
  return apiFetch<ConfirmReservationResponse>(`/reservations/${reservationId}/confirm`, {
    method: "POST",
  });
}

export function cancelReservation(
  reservationId: string,
): Promise<CancelReservationResponse> {
  return apiFetch<CancelReservationResponse>(`/reservations/${reservationId}/cancel`, {
    method: "POST",
  });
}

export function listMyActiveReservations(): Promise<ListMyActiveReservationsResponse> {
  return apiFetch<ListMyActiveReservationsResponse>("/reservations/my/active");
}
