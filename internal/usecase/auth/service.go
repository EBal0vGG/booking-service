package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TokenSigner signs JWT tokens for DummyLogin.
// We keep it interface-based for easy swapping in tests/config.
type TokenSigner interface {
	Sign(userID uuid.UUID, role domain.UserRole) (string, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

type bcryptHasher struct{}

func NewBcryptHasher() PasswordHasher {
	return &bcryptHasher{}
}

func (h *bcryptHasher) Hash(password string) (string, error) {
	out, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (h *bcryptHasher) Compare(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
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
	users  repository.UserRepository
	hasher PasswordHasher
	now    func() time.Time
}

func NewService(signer TokenSigner, users repository.UserRepository) *Service {
	return &Service{
		signer: signer,
		users:  users,
		hasher: NewBcryptHasher(),
		now:    func() time.Time { return time.Now().UTC() },
	}
}

var _ usecase.AuthUsecase = (*Service)(nil)

func (s *Service) Register(ctx context.Context, email, password string, role domain.UserRole) (domain.User, error) {
	if s.users == nil {
		return domain.User{}, domain.NewDomainError(domain.ErrorInternalError, "user repository is not configured")
	}

	email = normalizeEmail(email)
	if !isValidEmail(email) {
		return domain.User{}, domain.NewDomainError(domain.ErrorInvalidRequest, "invalid email")
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return domain.User{}, domain.NewDomainError(domain.ErrorInvalidRequest, "password is required")
	}
	if len(password) < 8 {
		return domain.User{}, domain.NewDomainError(domain.ErrorInvalidRequest, "password must be at least 8 characters")
	}
	if role != domain.RoleAdmin && role != domain.RoleUser {
		return domain.User{}, domain.NewDomainError(domain.ErrorInvalidRequest, "invalid role")
	}

	existing, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return domain.User{}, domain.WrapDomainError(domain.ErrorInternalError, "get user by email", err)
	}
	if existing != nil {
		// API contract for /register requires duplicate email as 400 INVALID_REQUEST.
		return domain.User{}, domain.NewDomainError(domain.ErrorInvalidRequest, "email already exists")
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, domain.WrapDomainError(domain.ErrorInternalError, "hash password", err)
	}

	user := domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &hash,
		Role:         role,
		CreatedAt:    s.now(),
	}
	if err := s.users.Create(ctx, user); err != nil {
		if _, ok := domain.AsDomainError(err); ok {
			// Repo may return INVALID_REQUEST (e.g. DB unique email conflict); pass through as-is.
			return domain.User{}, err
		}
		return domain.User{}, domain.WrapDomainError(domain.ErrorInternalError, "create user", err)
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	if s.users == nil {
		return "", domain.NewDomainError(domain.ErrorInternalError, "user repository is not configured")
	}

	email = normalizeEmail(email)
	if !isValidEmail(email) || strings.TrimSpace(password) == "" {
		return "", domain.NewDomainError(domain.ErrorUnauthorized, "invalid credentials")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return "", domain.WrapDomainError(domain.ErrorInternalError, "get user by email", err)
	}
	if user == nil || user.PasswordHash == nil {
		return "", domain.NewDomainError(domain.ErrorUnauthorized, "invalid credentials")
	}
	if err := s.hasher.Compare(*user.PasswordHash, password); err != nil {
		return "", domain.NewDomainError(domain.ErrorUnauthorized, "invalid credentials")
	}

	token, err := s.signer.Sign(user.ID, user.Role)
	if err != nil {
		return "", domain.WrapDomainError(domain.ErrorInternalError, "sign token", err)
	}
	return token, nil
}

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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Address == email
}
