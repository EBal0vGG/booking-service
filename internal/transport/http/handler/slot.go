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

	dateRaw := r.URL.Query().Get("date")
	if dateRaw == "" {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInvalidRequest, "date query parameter is required"))
		return
	}
	date, err := time.Parse("2006-01-02", dateRaw)
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid date", err))
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
