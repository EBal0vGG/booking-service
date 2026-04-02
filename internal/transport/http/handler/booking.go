package handler

import (
	"net/http"
	"strconv"
	"time"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"
	"booking-service/internal/transport/http/response"
	"booking-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type BookingHandler struct {
	uc usecase.BookingUsecase
}

func NewBookingHandler(uc usecase.BookingUsecase) *BookingHandler {
	return &BookingHandler{uc: uc}
}

type bookingDTO struct {
	ID             string  `json:"id"`
	SlotID         string  `json:"slotId"`
	UserID         string  `json:"userId"`
	Status         string  `json:"status"`
	ConferenceLink *string `json:"conferenceLink,omitempty"`
	CreatedAt      string  `json:"createdAt"`
}

type createBookingRequest struct {
	SlotID               string `json:"slotId"`
	CreateConferenceLink bool   `json:"createConferenceLink"`
}

type createBookingResponse struct {
	Booking bookingDTO `json:"booking"`
}

type listBookingsResponse struct {
	Bookings   []bookingDTO  `json:"bookings"`
	Pagination paginationDTO `json:"pagination"`
}

type listMyBookingsResponse struct {
	Bookings []bookingDTO `json:"bookings"`
}

type cancelBookingResponse struct {
	Booking bookingDTO `json:"booking"`
}

type paginationDTO struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "booking usecase is not configured"))
		return
	}

	var req createBookingRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		response.WriteError(w, err)
		return
	}
	slotID, err := uuid.Parse(req.SlotID)
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid slotId", err))
		return
	}

	booking, err := h.uc.CreateBooking(r.Context(), user, slotID, req.CreateConferenceLink)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, createBookingResponse{Booking: bookingToDTO(booking)})
}

func (h *BookingHandler) ListBookings(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "booking usecase is not configured"))
		return
	}

	page, pageSize, err := parsePagination(r)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	bookings, pagination, err := h.uc.ListBookings(r.Context(), user, page, pageSize)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	resp := listBookingsResponse{
		Bookings: make([]bookingDTO, 0, len(bookings)),
		Pagination: paginationDTO{
			Page:     pagination.Page,
			PageSize: pagination.PageSize,
			Total:    pagination.Total,
		},
	}
	for _, booking := range bookings {
		resp.Bookings = append(resp.Bookings, bookingToDTO(booking))
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

func (h *BookingHandler) ListMyBookings(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "booking usecase is not configured"))
		return
	}

	bookings, err := h.uc.ListMyBookings(r.Context(), user)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	resp := listMyBookingsResponse{Bookings: make([]bookingDTO, 0, len(bookings))}
	for _, booking := range bookings {
		resp.Bookings = append(resp.Bookings, bookingToDTO(booking))
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

func (h *BookingHandler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "booking usecase is not configured"))
		return
	}

	bookingID, err := uuid.Parse(chi.URLParam(r, "bookingId"))
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid bookingId", err))
		return
	}

	booking, err := h.uc.CancelBooking(r.Context(), user, bookingID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, cancelBookingResponse{Booking: bookingToDTO(booking)})
}

func parsePagination(r *http.Request) (page int, pageSize int, err error) {
	page = 1
	pageSize = 20

	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid page", err)
		}
	}
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid pageSize", err)
		}
	}

	if page < 1 || pageSize < 1 || pageSize > 100 {
		return 0, 0, domain.NewDomainError(domain.ErrorInvalidRequest, "page must be >= 1 and pageSize between 1 and 100")
	}

	return page, pageSize, nil
}

func bookingToDTO(booking domain.Booking) bookingDTO {
	return bookingDTO{
		ID:             booking.ID.String(),
		SlotID:         booking.SlotID.String(),
		UserID:         booking.UserID.String(),
		Status:         string(booking.Status),
		ConferenceLink: booking.ConferenceLink,
		CreatedAt:      booking.CreatedAt.UTC().Format(time.RFC3339),
	}
}
