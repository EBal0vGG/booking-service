package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"booking-service/internal/domain"
	"booking-service/internal/repository"
	authsvc "booking-service/internal/usecase/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type authTestUsersRepo struct {
	getByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	createFn     func(ctx context.Context, user domain.User) error
}

func (m *authTestUsersRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *authTestUsersRepo) Create(ctx context.Context, user domain.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}

var _ repository.UserRepository = (*authTestUsersRepo)(nil)

func TestDummyLogin_HappyPathAdmin(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	uc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret), &authTestUsersRepo{})
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
	uc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret), &authTestUsersRepo{})
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
	uc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret), &authTestUsersRepo{})
	h := NewAuthHandler(uc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"superadmin"}`))
	req.Header.Set("Content-Type", "application/json")

	h.DummyLogin(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	code := decodeErrorCode(t, rec)
	require.Equal(t, string(domain.ErrorInvalidRequest), code)
}

func TestRegister_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	uc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret), &authTestUsersRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, user domain.User) error {
			require.Equal(t, "user@example.com", user.Email)
			require.NotNil(t, user.PasswordHash)
			return nil
		},
	})
	h := NewAuthHandler(uc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"user@example.com","password":"strong-password","role":"user"}`))
	req.Header.Set("Content-Type", "application/json")

	h.Register(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestLogin_InvalidPassword_Returns401(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	hash, err := authsvc.NewBcryptHasher().Hash("right")
	require.NoError(t, err)

	uc := authsvc.NewService(authsvc.NewHMACJWTSigner(secret), &authTestUsersRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: uuid.New(), Email: email, Role: domain.RoleUser, PasswordHash: &hash}, nil
		},
	})
	h := NewAuthHandler(uc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"user@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Login(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	code := decodeErrorCode(t, rec)
	require.Equal(t, string(domain.ErrorUnauthorized), code)
}
