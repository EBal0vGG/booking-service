package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"booking-service/internal/domain"
	authsvc "booking-service/internal/usecase/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type apiErrResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func signHS256Token(secret string, userID uuid.UUID, role domain.UserRole, expUnix int64) (string, error) {
	header := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]any{
		"user_id": userID.String(),
		"role":    string(role),
		"exp":     expUnix,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := encHeader + "." + encPayload

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	signature := mac.Sum(nil)

	encSig := base64.RawURLEncoding.EncodeToString(signature)
	return unsigned + "." + encSig, nil
}

func TestRequireUser_NoBearer_Returns401(t *testing.T) {
	t.Parallel()

	auth := NewAuth("test-secret")
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	auth.RequireUser(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, called)

	var body apiErrResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, string(domain.ErrorUnauthorized), body.Error.Code)
}

func TestRequireUser_BadToken_Returns401(t *testing.T) {
	t.Parallel()

	auth := NewAuth("test-secret")
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	auth.RequireUser(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, called)

	var body apiErrResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, string(domain.ErrorUnauthorized), body.Error.Code)
}

func TestRequireUser_ExpiredToken_Returns401(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	auth := NewAuth(secret)

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	token, err := signHS256Token(secret, userID, domain.RoleUser, time.Now().UTC().Unix()-10)
	require.NoError(t, err)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	auth.RequireUser(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, called)

	var body apiErrResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, string(domain.ErrorUnauthorized), body.Error.Code)
}

func TestRequireUser_ValidToken_CallsNextAndSetsUser(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	auth := NewAuth(secret)

	svc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret))
	token, err := svc.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, 2, strings.Count(token, "."))

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		u, ok := UserFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, domain.RoleUser, u.Role)
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	auth.RequireUser(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, called)
}

