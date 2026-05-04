package realtime

import (
	"net/http"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"
	"booking-service/internal/transport/http/response"

	"github.com/gorilla/websocket"
)

type WSHandler struct {
	hub      *Hub
	auth     *authmw.Auth
	upgrader websocket.Upgrader
}

func NewWSHandler(hub *Hub, jwtSecret string) *WSHandler {
	return &WSHandler{
		hub:  hub,
		auth: authmw.NewAuth(jwtSecret),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: replace with explicit allowed-origins whitelist before production use.
				// Allow any origin for local/dev/test environment.
				return true
			},
		},
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticate(r)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(conn, h.hub, user)
	client.Run()
}

func (h *WSHandler) authenticate(r *http.Request) (domain.User, error) {
	// Preferred path: standard Authorization: Bearer <token>.
	if user, err := h.auth.AuthenticateRequest(r); err == nil {
		return user, nil
	}

	// Browser WS often cannot set custom auth headers; allow fallback query token.
	token := r.URL.Query().Get("token")
	if token == "" {
		return domain.User{}, domain.NewDomainError(domain.ErrorUnauthorized, "missing bearer token")
	}
	return h.auth.AuthenticateToken(token)
}
