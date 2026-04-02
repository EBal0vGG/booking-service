package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDummyLogin_AdminTokenNotEmpty(t *testing.T) {
	t.Parallel()

	svc := NewService(NewHMACJWTSigner("test-secret"))
	token, err := svc.DummyLogin(t.Context(), domain.RoleAdmin)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, 2, strings.Count(token, "."))
}

func TestDummyLogin_UserTokenNotEmpty(t *testing.T) {
	t.Parallel()

	svc := NewService(NewHMACJWTSigner("test-secret"))
	token, err := svc.DummyLogin(t.Context(), domain.RoleUser)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, 2, strings.Count(token, "."))
}

func TestDummyLogin_InvalidRole(t *testing.T) {
	t.Parallel()

	svc := NewService(NewHMACJWTSigner("test-secret"))
	_, err := svc.DummyLogin(t.Context(), domain.UserRole("superadmin"))

	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
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

