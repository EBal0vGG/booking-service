package handler

import (
	"net/http"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/transport/http/response"
	"booking-service/internal/usecase"
)

type AuthHandler struct {
	uc usecase.AuthUsecase
}

func NewAuthHandler(uc usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

type dummyLoginRequest struct {
	Role string `json:"role"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userDTO struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	CreatedAt *string `json:"createdAt,omitempty"`
}

type registerResponse struct {
	User userDTO `json:"user"`
}

func (h *AuthHandler) DummyLogin(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "auth usecase is not configured"))
		return
	}

	var req dummyLoginRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	if req.Role != "admin" && req.Role != "user" {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInvalidRequest, "invalid role"))
		return
	}

	token, err := h.uc.DummyLogin(r.Context(), domain.UserRole(req.Role))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "auth usecase is not configured"))
		return
	}

	var req registerRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	user, err := h.uc.Register(r.Context(), req.Email, req.Password, domain.UserRole(req.Role))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	resp := registerResponse{User: userToDTO(user)}
	response.WriteJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "auth usecase is not configured"))
		return
	}

	var req loginRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	token, err := h.uc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func userToDTO(user domain.User) userDTO {
	dto := userDTO{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  string(user.Role),
	}
	if !user.CreatedAt.IsZero() {
		ts := user.CreatedAt.UTC().Format(time.RFC3339)
		dto.CreatedAt = &ts
	}
	return dto
}
