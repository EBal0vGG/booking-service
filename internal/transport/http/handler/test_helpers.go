package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"booking-service/internal/domain"
	authsvc "booking-service/internal/usecase/auth"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type apiErrResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func mustJWT(t *testing.T, secret string, role domain.UserRole) string {
	t.Helper()

	svc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret))
	token, err := svc.DummyLogin(context.Background(), role)
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimSpace(token))
	return token
}

func setChiParam(req *http.Request, key, value string) *http.Request {
	req = req.Clone(req.Context())
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func decodeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body apiErrResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Error.Code
}

func decodeErrorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body apiErrResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Error.Message
}

