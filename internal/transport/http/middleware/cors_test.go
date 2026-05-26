package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCORSOptionsRequestReturnsNoContent(t *testing.T) {
	t.Parallel()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/register", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	CORS([]string{"http://localhost:3000"})(next).ServeHTTP(rec, req)

	require.False(t, called)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, corsAllowMethods, rec.Header().Get("Access-Control-Allow-Methods"))
	require.Equal(t, corsAllowHeaders, rec.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORSNonOptionsPassesThrough(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	CORS([]string{"http://localhost:3000"})(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, corsExposeHeader, rec.Header().Get("Access-Control-Expose-Headers"))
}

func TestCORSDisallowedOriginDoesNotExposeHeaders(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	req.Header.Set("Origin", "http://evil.local")
	rec := httptest.NewRecorder()

	CORS([]string{"http://localhost:3000"})(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}
