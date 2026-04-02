package handler

import (
	"net/http"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"
	"booking-service/internal/transport/http/response"
	"booking-service/internal/usecase"
)

type RoomHandler struct {
	uc usecase.RoomUsecase
}

func NewRoomHandler(uc usecase.RoomUsecase) *RoomHandler {
	return &RoomHandler{uc: uc}
}

type roomDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Capacity    *int    `json:"capacity,omitempty"`
}

type listRoomsResponse struct {
	Rooms []roomDTO `json:"rooms"`
}

type createRoomRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Capacity    *int    `json:"capacity,omitempty"`
}

type createRoomResponse struct {
	Room roomDTO `json:"room"`
}

func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "room usecase is not configured"))
		return
	}

	rooms, err := h.uc.ListRooms(r.Context(), user)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	resp := listRoomsResponse{Rooms: make([]roomDTO, 0, len(rooms))}
	for _, room := range rooms {
		resp.Rooms = append(resp.Rooms, roomToDTO(room))
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "room usecase is not configured"))
		return
	}

	var req createRoomRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	room, err := h.uc.CreateRoom(r.Context(), user, usecase.RoomCreateInput{
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, createRoomResponse{Room: roomToDTO(room)})
}

func roomToDTO(room domain.Room) roomDTO {
	return roomDTO{
		ID:          room.ID.String(),
		Name:        room.Name,
		Description: room.Description,
		Capacity:    room.Capacity,
	}
}
