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

type SlotHandler struct {
	uc usecase.SlotUsecase
}

func NewSlotHandler(uc usecase.SlotUsecase) *SlotHandler {
	return &SlotHandler{uc: uc}
}

type slotDTO struct {
	ID      string `json:"id"`
	RoomID  string `json:"roomId"`
	StartAt string `json:"start"`
	EndAt   string `json:"end"`
}

type listSlotsResponse struct {
	Slots []slotDTO `json:"slots"`
}

type slotViewDTO struct {
	ID              string  `json:"id"`
	RoomID          string  `json:"roomId"`
	StartAt         string  `json:"start"`
	EndAt           string  `json:"end"`
	Status          string  `json:"status"`
	BookingID       *string `json:"bookingId,omitempty"`
	ReservationID   *string `json:"reservationId,omitempty"`
	WaitlistEntryID *string `json:"waitlistEntryId,omitempty"`
}

type listSlotViewsResponse struct {
	Slots []slotViewDTO `json:"slots"`
}

func (h *SlotHandler) ListAvailableSlots(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "slot usecase is not configured"))
		return
	}

	roomID, err := uuid.Parse(chi.URLParam(r, "roomId"))
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid roomId", err))
		return
	}

	date, err := parseDateQuery(r)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	slots, err := h.uc.ListAvailableSlots(r.Context(), user, roomID, date)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	resp := listSlotsResponse{Slots: make([]slotDTO, 0, len(slots))}
	for _, slot := range slots {
		resp.Slots = append(resp.Slots, slotDTO{
			ID:      slot.ID.String(),
			RoomID:  slot.RoomID.String(),
			StartAt: slot.StartTime.UTC().Format(time.RFC3339),
			EndAt:   slot.EndTime.UTC().Format(time.RFC3339),
		})
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

func (h *SlotHandler) ListRoomSlots(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "slot usecase is not configured"))
		return
	}

	roomID, err := uuid.Parse(chi.URLParam(r, "roomId"))
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid roomId", err))
		return
	}

	date, err := parseDateQuery(r)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	slots, err := h.uc.ListRoomSlots(r.Context(), user, roomID, date)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	resp := listSlotViewsResponse{Slots: make([]slotViewDTO, 0, len(slots))}
	for _, slot := range slots {
		dto := slotViewDTO{
			ID:      slot.ID.String(),
			RoomID:  slot.RoomID.String(),
			StartAt: slot.StartTime.UTC().Format(time.RFC3339),
			EndAt:   slot.EndTime.UTC().Format(time.RFC3339),
			Status:  string(slot.Status),
		}
		if slot.BookingID != nil {
			value := slot.BookingID.String()
			dto.BookingID = &value
		}
		if slot.ReservationID != nil {
			value := slot.ReservationID.String()
			dto.ReservationID = &value
		}
		if slot.WaitlistEntryID != nil {
			value := slot.WaitlistEntryID.String()
			dto.WaitlistEntryID = &value
		}
		resp.Slots = append(resp.Slots, dto)
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

func parseDateQuery(r *http.Request) (time.Time, error) {
	dateRaw := r.URL.Query().Get("date")
	if dateRaw == "" {
		return time.Time{}, domain.NewDomainError(domain.ErrorInvalidRequest, "date query parameter is required")
	}
	date, err := time.Parse("2006-01-02", dateRaw)
	if err != nil {
		return time.Time{}, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid date", err)
	}
	return date, nil
}
