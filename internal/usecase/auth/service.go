package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"booking-service/internal/domain"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
)

// TokenSigner signs JWT tokens for DummyLogin.
// We keep it interface-based for easy swapping in tests/config.
type TokenSigner interface {
	Sign(userID uuid.UUID, role domain.UserRole) (string, error)
}

type hmacJWTSigner struct {
	secret []byte
}

func NewHMACJWTSigner(secret string) TokenSigner {
	return &hmacJWTSigner{secret: []byte(secret)}
}

func (s *hmacJWTSigner) Sign(userID uuid.UUID, role domain.UserRole) (string, error) {
	header := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}

	exp := time.Now().UTC().Add(24 * time.Hour).Unix()
	payload := map[string]any{
		"user_id": userID.String(),
		"role":    string(role),
		"exp":     exp,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal jwt payload: %w", err)
	}

	encHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := encHeader + "." + encPayload

	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(unsigned))
	signature := mac.Sum(nil)

	encSig := base64.RawURLEncoding.EncodeToString(signature)
	return unsigned + "." + encSig, nil
}

type Service struct {
	signer TokenSigner
}

func NewService(signer TokenSigner) *Service {
	return &Service{signer: signer}
}

var _ usecase.AuthUsecase = (*Service)(nil)

func (s *Service) DummyLogin(ctx context.Context, role domain.UserRole) (string, error) {
	_ = ctx // reserved for future (e.g. auditing/logging)

	var userID uuid.UUID
	switch role {
	case domain.RoleAdmin:
		userID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	case domain.RoleUser:
		userID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	default:
		return "", domain.NewDomainError(domain.ErrorInvalidRequest, "invalid role")
	}

	return s.signer.Sign(userID, role)
}

