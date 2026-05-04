package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/transport/http/response"

	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "auth_user"

type Auth struct {
	secret []byte
}

func NewAuth(secret string) *Auth {
	return &Auth{secret: []byte(secret)}
}

func (a *Auth) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.AuthenticateRequest(r)
		if err != nil {
			response.WriteError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) AuthenticateRequest(r *http.Request) (domain.User, error) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return domain.User{}, domain.NewDomainError(domain.ErrorUnauthorized, "missing bearer token")
	}
	return a.AuthenticateToken(token)
}

func (a *Auth) AuthenticateToken(token string) (domain.User, error) {
	return a.parseAndValidateToken(token)
}

func UserFromContext(ctx context.Context) (domain.User, bool) {
	v := ctx.Value(userContextKey)
	user, ok := v.(domain.User)
	return user, ok
}

func bearerToken(authHeader string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
}

type jwtClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

func (a *Auth) parseAndValidateToken(token string) (domain.User, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return domain.User{}, domain.NewDomainError(domain.ErrorUnauthorized, "invalid token format")
	}

	unsigned := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return domain.User{}, domain.WrapDomainError(domain.ErrorUnauthorized, "invalid token signature", err)
	}

	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(unsigned))
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return domain.User{}, domain.NewDomainError(domain.ErrorUnauthorized, "invalid token signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return domain.User{}, domain.WrapDomainError(domain.ErrorUnauthorized, "invalid token payload", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return domain.User{}, domain.WrapDomainError(domain.ErrorUnauthorized, "invalid token claims", err)
	}
	if claims.Exp > 0 && time.Now().UTC().Unix() >= claims.Exp {
		return domain.User{}, domain.NewDomainError(domain.ErrorUnauthorized, "token is expired")
	}

	userID, err := parseUserID(claims.UserID)
	if err != nil {
		return domain.User{}, err
	}

	role := domain.UserRole(claims.Role)
	if role != domain.RoleAdmin && role != domain.RoleUser {
		return domain.User{}, domain.NewDomainError(domain.ErrorUnauthorized, "invalid role claim")
	}

	return domain.User{
		ID:    userID,
		Email: "",
		Role:  role,
	}, nil
}

func parseUserID(raw string) (id uuid.UUID, err error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return id, domain.WrapDomainError(domain.ErrorUnauthorized, "invalid user id in token", err)
	}
	return parsed, nil
}
