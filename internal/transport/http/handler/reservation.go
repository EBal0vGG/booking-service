package handler

import (
	"net/http"
	"time"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"
	"booking-service/internal/transport/http/response"
	"booking-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ReservationHandler struct {
	uc usecase.ReservationUsecase
}

func NewReservationHandler(uc usecase.ReservationUsecase) *ReservationHandler {
	return &ReservationHandler{uc: uc}
}

type reservationDTO struct {
	ID              string  `json:"id"`
	SlotID          string  `json:"slotId"`
	UserID          string  `json:"userId"`
	WaitlistEntryID *string `json:"waitlistEntryId,omitempty"`
	Status          string  `json:"status"`
	ExpiresAt       string  `json:"expiresAt"`
	CreatedAt       string  `json:"createdAt"`
	ConfirmedAt     *string `json:"confirmedAt,omitempty"`
	ExpiredAt       *string `json:"expiredAt,omitempty"`
}

type confirmReservationResponse struct {
	Booking     bookingDTO     `json:"booking"`
	Reservation reservationDTO `json:"reservation"`
}

type cancelReservationResponse struct {
	Reservation reservationDTO `json:"reservation"`
}

type listMyActiveReservationsResponse struct {
	Reservations []reservationDTO `json:"reservations"`
}

func (h *ReservationHandler) ConfirmReservation(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "reservation usecase is not configured"))
		return
	}

	reservationID, err := uuid.Parse(chi.URLParam(r, "reservationId"))
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid reservationId", err))
		return
	}

	booking, reservation, err := h.uc.ConfirmReservation(r.Context(), user, reservationID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, confirmReservationResponse{
		Booking:     bookingToDTO(booking),
		Reservation: reservationToDTO(reservation),
	})
}

func (h *ReservationHandler) CancelReservation(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "reservation usecase is not configured"))
		return
	}

	reservationID, err := uuid.Parse(chi.URLParam(r, "reservationId"))
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid reservationId", err))
		return
	}

	reservation, err := h.uc.CancelReservation(r.Context(), user, reservationID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, cancelReservationResponse{
		Reservation: reservationToDTO(reservation),
	})
}

func (h *ReservationHandler) ListMyActiveReservations(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "reservation usecase is not configured"))
		return
	}

	reservations, err := h.uc.ListMyActiveReservations(r.Context(), user)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	resp := listMyActiveReservationsResponse{
		Reservations: make([]reservationDTO, 0, len(reservations)),
	}
	for _, reservation := range reservations {
		resp.Reservations = append(resp.Reservations, reservationToDTO(reservation))
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

func reservationToDTO(reservation domain.SlotReservation) reservationDTO {
	dto := reservationDTO{
		ID:        reservation.ID.String(),
		SlotID:    reservation.SlotID.String(),
		UserID:    reservation.UserID.String(),
		Status:    string(reservation.Status),
		ExpiresAt: reservation.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt: reservation.CreatedAt.UTC().Format(time.RFC3339),
	}
	if reservation.WaitlistEntryID != nil {
		value := reservation.WaitlistEntryID.String()
		dto.WaitlistEntryID = &value
	}
	if reservation.ConfirmedAt != nil {
		value := reservation.ConfirmedAt.UTC().Format(time.RFC3339)
		dto.ConfirmedAt = &value
	}
	if reservation.ExpiredAt != nil {
		value := reservation.ExpiredAt.UTC().Format(time.RFC3339)
		dto.ExpiredAt = &value
	}
	return dto
}
