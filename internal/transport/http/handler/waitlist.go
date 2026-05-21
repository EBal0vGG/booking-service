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

type WaitlistHandler struct {
	uc usecase.WaitlistUsecase
}

func NewWaitlistHandler(uc usecase.WaitlistUsecase) *WaitlistHandler {
	return &WaitlistHandler{uc: uc}
}

type joinWaitlistRequest struct {
	SlotID string `json:"slotId"`
}

type waitlistEntryDTO struct {
	ID         string  `json:"id"`
	SlotID     string  `json:"slotId"`
	UserID     string  `json:"userId"`
	Status     string  `json:"status"`
	Position   int64   `json:"position"`
	CreatedAt  string  `json:"createdAt"`
	NotifiedAt *string `json:"notifiedAt,omitempty"`
}

type joinWaitlistResponse struct {
	Entry waitlistEntryDTO `json:"entry"`
}

type leaveWaitlistResponse struct {
	Entry waitlistEntryDTO `json:"entry"`
}

func (h *WaitlistHandler) JoinWaitlist(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "waitlist usecase is not configured"))
		return
	}

	var req joinWaitlistRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	slotID, err := uuid.Parse(req.SlotID)
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid slotId", err))
		return
	}

	entry, err := h.uc.JoinWaitlist(r.Context(), user, slotID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, joinWaitlistResponse{Entry: waitlistEntryToDTO(entry)})
}

func (h *WaitlistHandler) LeaveWaitlist(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "waitlist usecase is not configured"))
		return
	}

	entryID, err := uuid.Parse(chi.URLParam(r, "waitlistId"))
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid waitlistId", err))
		return
	}

	entry, err := h.uc.LeaveWaitlist(r.Context(), user, entryID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, leaveWaitlistResponse{Entry: waitlistEntryToDTO(entry)})
}

func waitlistEntryToDTO(entry domain.WaitlistEntry) waitlistEntryDTO {
	dto := waitlistEntryDTO{
		ID:        entry.ID.String(),
		SlotID:    entry.SlotID.String(),
		UserID:    entry.UserID.String(),
		Status:    string(entry.Status),
		Position:  entry.Position,
		CreatedAt: entry.CreatedAt.UTC().Format(time.RFC3339),
	}
	if entry.NotifiedAt != nil {
		v := entry.NotifiedAt.UTC().Format(time.RFC3339)
		dto.NotifiedAt = &v
	}
	return dto
}
