package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockWaitlistUC struct {
	joinFn  func(ctx context.Context, user domain.User, slotID uuid.UUID) (domain.WaitlistEntry, error)
	leaveFn func(ctx context.Context, user domain.User, entryID uuid.UUID) (domain.WaitlistEntry, error)
}

func (m *mockWaitlistUC) JoinWaitlist(ctx context.Context, user domain.User, slotID uuid.UUID) (domain.WaitlistEntry, error) {
	return m.joinFn(ctx, user, slotID)
}

func (m *mockWaitlistUC) LeaveWaitlist(ctx context.Context, user domain.User, entryID uuid.UUID) (domain.WaitlistEntry, error) {
	return m.leaveFn(ctx, user, entryID)
}

func TestJoinWaitlist_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)
	slotID := uuid.New()
	entryID := uuid.New()
	userID := uuid.New()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	uc := &mockWaitlistUC{
		joinFn: func(ctx context.Context, user domain.User, gotSlotID uuid.UUID) (domain.WaitlistEntry, error) {
			require.Equal(t, slotID, gotSlotID)
			return domain.WaitlistEntry{
				ID:        entryID,
				SlotID:    slotID,
				UserID:    userID,
				Status:    domain.WaitlistStatusActive,
				Position:  42,
				CreatedAt: now,
			}, nil
		},
	}

	h := NewWaitlistHandler(uc)
	wrapped := authmw.NewAuth(secret).RequireUser(http.HandlerFunc(h.JoinWaitlist))

	req := httptest.NewRequest(http.MethodPost, "/waitlist/join", strings.NewReader(`{"slotId":"`+slotID.String()+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var out joinWaitlistResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, entryID.String(), out.Entry.ID)
	require.Equal(t, int64(42), out.Entry.Position)
}

func TestJoinWaitlist_DuplicateMappedTo409(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)
	slotID := uuid.New()
	uc := &mockWaitlistUC{
		joinFn: func(ctx context.Context, user domain.User, slotID uuid.UUID) (domain.WaitlistEntry, error) {
			return domain.WaitlistEntry{}, domain.NewDomainError(domain.ErrorWaitlistJoined, "already joined")
		},
	}
	h := NewWaitlistHandler(uc)
	wrapped := authmw.NewAuth(secret).RequireUser(http.HandlerFunc(h.JoinWaitlist))

	req := httptest.NewRequest(http.MethodPost, "/waitlist/join", strings.NewReader(`{"slotId":"`+slotID.String()+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, string(domain.ErrorWaitlistJoined), decodeErrorCode(t, rec))
}

func TestLeaveWaitlist_InvalidID_Returns400(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)
	uc := &mockWaitlistUC{
		leaveFn: func(ctx context.Context, user domain.User, entryID uuid.UUID) (domain.WaitlistEntry, error) {
			t.Fatal("must not be called")
			return domain.WaitlistEntry{}, nil
		},
	}
	h := NewWaitlistHandler(uc)
	wrapped := authmw.NewAuth(secret).RequireUser(http.HandlerFunc(h.LeaveWaitlist))

	req := httptest.NewRequest(http.MethodPost, "/waitlist/not-uuid/leave", nil)
	req = setChiParam(req, "waitlistId", "not-uuid")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, string(domain.ErrorInvalidRequest), decodeErrorCode(t, rec))
}
