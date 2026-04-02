package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"booking-service/internal/domain"
	authsvc "booking-service/internal/usecase/auth"

	"github.com/stretchr/testify/require"
)

func TestDummyLogin_HappyPathAdmin(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	uc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret))
	h := NewAuthHandler(uc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")

	h.DummyLogin(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var out tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.NotEmpty(t, strings.TrimSpace(out.Token))
}

func TestDummyLogin_HappyPathUser(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	uc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret))
	h := NewAuthHandler(uc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"user"}`))
	req.Header.Set("Content-Type", "application/json")

	h.DummyLogin(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var out tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.NotEmpty(t, strings.TrimSpace(out.Token))
}

func TestDummyLogin_InvalidRole_Returns400(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	uc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret))
	h := NewAuthHandler(uc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"superadmin"}`))
	req.Header.Set("Content-Type", "application/json")

	h.DummyLogin(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	code := decodeErrorCode(t, rec)
	require.Equal(t, string(domain.ErrorInvalidRequest), code)
}

