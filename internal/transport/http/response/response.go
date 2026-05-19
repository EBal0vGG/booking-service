package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"booking-service/internal/domain"
)

// MaxBody limits JSON request body size (1 MiB).
const MaxBody = 1 << 20

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// WriteJSON writes JSON with Content-Type application/json.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteError maps errors to JSON API responses. Non-domain errors become INTERNAL_ERROR.
// Always logs the underlying error for observability.
func WriteError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	slog.Error("http error response", "error", err)

	if de, ok := domain.AsDomainError(err); ok {
		body := errorBody{}
		body.Error.Code = string(de.Code)
		body.Error.Message = de.Message
		WriteJSON(w, StatusFromDomainCode(de.Code), body)
		return
	}

	body := errorBody{}
	body.Error.Code = string(domain.ErrorInternalError)
	body.Error.Message = "internal server error"
	WriteJSON(w, http.StatusInternalServerError, body)
}

// StatusFromDomainCode maps domain error codes to HTTP status.
func StatusFromDomainCode(code domain.ErrorCode) int {
	switch code {
	case domain.ErrorInvalidRequest:
		return http.StatusBadRequest
	case domain.ErrorUnauthorized:
		return http.StatusUnauthorized
	case domain.ErrorForbidden:
		return http.StatusForbidden
	case domain.ErrorNotFound, domain.ErrorRoomNotFound, domain.ErrorSlotNotFound, domain.ErrorBookingNotFound:
		return http.StatusNotFound
	case domain.ErrorSlotAlreadyBooked, domain.ErrorScheduleExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// DecodeJSON decodes JSON from r.Body with a size limit.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	limited := http.MaxBytesReader(w, r.Body, MaxBody)
	dec := json.NewDecoder(limited)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return domain.NewDomainError(domain.ErrorInvalidRequest, "request body too large")
		}
		return domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid json body", err)
	}
	return nil
}
