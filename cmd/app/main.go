package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"booking-service/internal/app"
	"booking-service/internal/config"
	"booking-service/internal/realtime"
	"booking-service/internal/repository/postgres"
	httptransport "booking-service/internal/transport/http"
	authsvc "booking-service/internal/usecase/auth"
	bookinguc "booking-service/internal/usecase/booking"
	roomuc "booking-service/internal/usecase/room"
	scheduleuc "booking-service/internal/usecase/schedule"
	slotuc "booking-service/internal/usecase/slot"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPool, err := postgres.NewPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer dbPool.Close()

	if cfg.RunMigrations {
		if err := app.RunMigrations(cfg.MigrationsPath, cfg.DatabaseURL()); err != nil {
			log.Fatalf("run migrations: %v", err)
		}
	}

	roomRepo := postgres.NewRoomRepo(dbPool)
	userRepo := postgres.NewUserRepo(dbPool)
	scheduleRepo := postgres.NewScheduleRepo(dbPool)
	slotRepo := postgres.NewSlotRepo(dbPool)
	bookingRepo := postgres.NewBookingRepo(dbPool)
	txManager := postgres.NewTxManager(dbPool)
	realtimeHub := realtime.NewHub()
	realtimeManager := realtime.NewManager(realtimeHub)
	wsHandler := realtime.NewWSHandler(realtimeHub, cfg.JWTSecret)

	router := httptransport.NewRouterWithDependencies(httptransport.RouterDependencies{
		AuthUC:     authsvc.NewService(authsvc.NewHMACJWTSigner(cfg.JWTSecret), userRepo),
		RoomUC:     roomuc.NewService(roomRepo),
		ScheduleUC: scheduleuc.NewService(roomRepo, scheduleRepo),
		SlotUC:     slotuc.NewService(roomRepo, slotRepo),
		BookingUC:  bookinguc.NewService(txManager, bookingRepo, slotRepo, realtimeManager),
		WSHandler:  wsHandler,
		JWTSecret:  cfg.JWTSecret,
	})
	server := httptransport.NewServer(cfg.Port, router)
	errCh := make(chan error, 1)

	go func() {
		log.Printf("HTTP server listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		log.Fatalf("http server failed: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
