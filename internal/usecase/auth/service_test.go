package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockUsersRepo struct {
	getByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	createFn     func(ctx context.Context, user domain.User) error
}

func (m *mockUsersRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockUsersRepo) Create(ctx context.Context, user domain.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}

var _ repository.UserRepository = (*mockUsersRepo)(nil)

func TestDummyLogin_AdminTokenNotEmpty(t *testing.T) {
	t.Parallel()

	svc := NewService(NewHMACJWTSigner("test-secret"), &mockUsersRepo{})
	token, err := svc.DummyLogin(t.Context(), domain.RoleAdmin)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, 2, strings.Count(token, "."))
}

func TestDummyLogin_UserTokenNotEmpty(t *testing.T) {
	t.Parallel()

	svc := NewService(NewHMACJWTSigner("test-secret"), &mockUsersRepo{})
	token, err := svc.DummyLogin(t.Context(), domain.RoleUser)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, 2, strings.Count(token, "."))
}

func TestDummyLogin_InvalidRole(t *testing.T) {
	t.Parallel()

	svc := NewService(NewHMACJWTSigner("test-secret"), &mockUsersRepo{})
	_, err := svc.DummyLogin(t.Context(), domain.UserRole("superadmin"))

	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestRegister_Success(t *testing.T) {
	t.Parallel()

	repo := &mockUsersRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, user domain.User) error {
			require.Equal(t, "user@example.com", user.Email)
			require.Equal(t, domain.RoleUser, user.Role)
			require.NotNil(t, user.PasswordHash)
			require.NotEmpty(t, *user.PasswordHash)
			return nil
		},
	}
	svc := NewService(NewHMACJWTSigner("test-secret"), repo)
	user, err := svc.Register(t.Context(), "User@Example.com", "strong-password", domain.RoleUser)
	require.NoError(t, err)
	require.Equal(t, "user@example.com", user.Email)
	require.Equal(t, domain.RoleUser, user.Role)
	require.NotNil(t, user.PasswordHash)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	t.Parallel()

	repo := &mockUsersRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: uuid.New(), Email: email, Role: domain.RoleUser}, nil
		},
	}
	svc := NewService(NewHMACJWTSigner("test-secret"), repo)
	_, err := svc.Register(t.Context(), "user@example.com", "strong-password", domain.RoleUser)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestRegister_TooShortPassword(t *testing.T) {
	t.Parallel()

	repo := &mockUsersRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			t.Fatal("should not query repo for invalid password")
			return nil, nil
		},
	}
	svc := NewService(NewHMACJWTSigner("test-secret"), repo)
	_, err := svc.Register(t.Context(), "user@example.com", "1234567", domain.RoleUser)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestLogin_InvalidPassword_ReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	hash, err := NewBcryptHasher().Hash("correct-password")
	require.NoError(t, err)

	repo := &mockUsersRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID:           uuid.New(),
				Email:        "user@example.com",
				PasswordHash: &hash,
				Role:         domain.RoleUser,
			}, nil
		},
	}
	svc := NewService(NewHMACJWTSigner("test-secret"), repo)
	_, err = svc.Login(t.Context(), "user@example.com", "wrong-password")
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorUnauthorized, de.Code)
}

func TestLogin_Success_ReturnsToken(t *testing.T) {
	t.Parallel()

	hash, err := NewBcryptHasher().Hash("correct-password")
	require.NoError(t, err)

	repo := &mockUsersRepo{
		getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID:           uuid.MustParse("22222222-2222-2222-2222-222222222222"),
				Email:        "user@example.com",
				PasswordHash: &hash,
				Role:         domain.RoleUser,
			}, nil
		},
	}
	svc := NewService(NewHMACJWTSigner("test-secret"), repo)
	token, err := svc.Login(t.Context(), "user@example.com", "correct-password")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, 2, strings.Count(token, "."))
}

func TestHMACJWTSigner_Sign_PayloadContainsRoleAndUserID(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	signer := NewHMACJWTSigner(secret)
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	token, err := signer.Sign(userID, domain.RoleUser)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(payloadBytes, &claims))

	require.Equal(t, userID.String(), claims["user_id"])
	require.Equal(t, string(domain.RoleUser), claims["role"])
	require.NotNil(t, claims["exp"])

	// Basic signature sanity: recompute expected signature.
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	expectedSig := mac.Sum(nil)
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	require.True(t, hmac.Equal(gotSig, expectedSig))

	// exp should be in the future (set to now+24h by signer).
	exp := int64(claims["exp"].(float64))
	require.Greater(t, exp, time.Now().UTC().Unix())
}
