package handler

import (
	"net/http"

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
