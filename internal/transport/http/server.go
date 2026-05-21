package http

import (
	"net/http"

	"booking-service/internal/transport/http/handler"
	authmw "booking-service/internal/transport/http/middleware"
	"booking-service/internal/transport/http/response"
	"booking-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}
}

type RouterDependencies struct {
	AuthUC     usecase.AuthUsecase
	RoomUC     usecase.RoomUsecase
	ScheduleUC usecase.ScheduleUsecase
	SlotUC     usecase.SlotUsecase
	BookingUC  usecase.BookingUsecase
	WaitlistUC usecase.WaitlistUsecase
	WSHandler  http.Handler
	JWTSecret  string
}

func NewRouterWithDependencies(deps RouterDependencies) chi.Router {
	router := chi.NewRouter()
	registerRoutes(router, deps)

	return router
}

func registerRoutes(r chi.Router, deps RouterDependencies) {
	auth := authmw.NewAuth(deps.JWTSecret)
	authHandler := handler.NewAuthHandler(deps.AuthUC)
	roomHandler := handler.NewRoomHandler(deps.RoomUC)
	scheduleHandler := handler.NewScheduleHandler(deps.ScheduleUC)
	slotHandler := handler.NewSlotHandler(deps.SlotUC)
	bookingHandler := handler.NewBookingHandler(deps.BookingUC)
	waitlistHandler := handler.NewWaitlistHandler(deps.WaitlistUC)

	r.Use(authmw.RequestID)
	r.Use(authmw.Recovery)
	r.Use(authmw.HTTPMetrics)

	r.Group(func(pub chi.Router) {
		pub.Use(authmw.RequestLogger)

		pub.Get("/_info", func(w http.ResponseWriter, _ *http.Request) {
			response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		pub.Handle("/metrics", promhttp.Handler())
		if deps.WSHandler != nil {
			pub.Handle("/ws", deps.WSHandler)
		}

		pub.Post("/register", authHandler.Register)
		pub.Post("/login", authHandler.Login)
		pub.Post("/dummyLogin", authHandler.DummyLogin)
	})

	r.Group(func(pr chi.Router) {
		pr.Use(auth.RequireUser)
		pr.Use(authmw.RequestLogger)

		pr.Get("/rooms/list", roomHandler.ListRooms)
		pr.Post("/rooms/create", roomHandler.CreateRoom)
		pr.Post("/rooms/{roomId}/schedule/create", scheduleHandler.CreateSchedule)
		pr.Get("/rooms/{roomId}/slots/list", slotHandler.ListAvailableSlots)

		pr.Post("/bookings/create", bookingHandler.CreateBooking)
		pr.Get("/bookings/list", bookingHandler.ListBookings)
		pr.Get("/bookings/my", bookingHandler.ListMyBookings)
		pr.Post("/bookings/{bookingId}/cancel", bookingHandler.CancelBooking)

		pr.Post("/waitlist/join", waitlistHandler.JoinWaitlist)
		pr.Post("/waitlist/{waitlistId}/leave", waitlistHandler.LeaveWaitlist)
	})
}
