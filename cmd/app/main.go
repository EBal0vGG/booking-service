package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"booking-service/internal/app"
	"booking-service/internal/app/slotgen"
	"booking-service/internal/config"
	"booking-service/internal/observability/logging"
	observabilitymetrics "booking-service/internal/observability/metrics"
	"booking-service/internal/realtime"
	"booking-service/internal/repository/postgres"
	httptransport "booking-service/internal/transport/http"
	authsvc "booking-service/internal/usecase/auth"
	bookinguc "booking-service/internal/usecase/booking"
	reservationuc "booking-service/internal/usecase/reservation"
	roomuc "booking-service/internal/usecase/room"
	scheduleuc "booking-service/internal/usecase/schedule"
	slotuc "booking-service/internal/usecase/slot"
	waitlistuc "booking-service/internal/usecase/waitlist"

	redis "github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		return
	}
	logger := logging.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPool, err := postgres.NewPool(ctx, cfg.DatabaseURL())
	if err != nil {
		slog.Error("connect postgres", "error", err)
		return
	}
	defer dbPool.Close()

	if cfg.RunMigrations {
		if err := app.RunMigrations(cfg.MigrationsPath, cfg.DatabaseURL()); err != nil {
			slog.Error("run migrations", "error", err)
			return
		}
	}

	roomRepo := postgres.NewRoomRepo(dbPool)
	userRepo := postgres.NewUserRepo(dbPool)
	scheduleRepo := postgres.NewScheduleRepo(dbPool)
	slotRepo := postgres.NewSlotRepo(dbPool)
	bookingRepo := postgres.NewBookingRepo(dbPool)
	waitlistRepo := postgres.NewWaitlistRepo(dbPool)
	reservationRepo := postgres.NewReservationRepo(dbPool)
	txManager := postgres.NewTxManager(dbPool)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	pingCtx, pingCancel := context.WithTimeout(ctx, time.Second)
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		slog.Warn("redis unavailable at startup; realtime in degraded mode until reconnect", "addr", cfg.RedisAddr, "error", err)
	} else {
		slog.Info("redis connected", "addr", cfg.RedisAddr, "db", cfg.RedisDB, "channel", cfg.RedisChannel)
	}
	pingCancel()
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Warn("close redis client", "error", err)
		}
	}()

	realtimeHub := realtime.NewHub()
	realtimePublisher := realtime.NewRedisPublisher(redisClient, cfg.RedisChannel)
	realtimeSubscriber := realtime.NewRedisSubscriber(redisClient, cfg.RedisChannel, realtimeHub)
	wsHandler := realtime.NewWSHandler(realtimeHub, cfg.JWTSecret)
	bookingService := bookinguc.NewService(txManager, bookingRepo, slotRepo, realtimePublisher).
		WithWaitlistRepository(waitlistRepo).
		WithReservationRepository(reservationRepo, cfg.ReservationTTL).
		WithMetrics(observabilitymetrics.NewBookingUsecaseMetrics())
	waitlistService := waitlistuc.NewService(txManager, waitlistRepo, slotRepo, bookingRepo)
	reservationService := reservationuc.NewService(
		txManager,
		reservationRepo,
		bookingRepo,
		slotRepo,
		waitlistRepo,
		realtimePublisher,
		cfg.ReservationTTL,
	).WithMetrics(observabilitymetrics.NewReservationUsecaseMetrics())
	slotGenerator := slotgen.NewGenerator(roomRepo, scheduleRepo, slotRepo)

	var backgroundWG sync.WaitGroup
	subscriberErrCh := make(chan error, 1)
	backgroundWG.Add(1)
	go func() {
		defer backgroundWG.Done()
		if err := realtimeSubscriber.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			subscriberErrCh <- err
		}
	}()
	backgroundWG.Add(1)
	go func() {
		defer backgroundWG.Done()
		ticker := time.NewTicker(cfg.ReservationExpireInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := reservationService.ExpireDue(ctx, cfg.ReservationExpireBatch); err != nil && !errors.Is(err, context.Canceled) {
					slog.Warn("reservation expiration worker failed", "error", err)
				}
			}
		}
	}()
	backgroundWG.Add(1)
	go func() {
		defer backgroundWG.Done()

		if err := slotGenerator.Generate(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Warn("slot generation failed", "error", err)
		}

		ticker := time.NewTicker(cfg.SlotGenerateInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := slotGenerator.Generate(ctx); err != nil && !errors.Is(err, context.Canceled) {
					slog.Warn("slot generation failed", "error", err)
				}
			}
		}
	}()

	router := httptransport.NewRouterWithDependencies(httptransport.RouterDependencies{
		AuthUC:        authsvc.NewService(authsvc.NewHMACJWTSigner(cfg.JWTSecret), userRepo),
		RoomUC:        roomuc.NewService(roomRepo),
		ScheduleUC:    scheduleuc.NewService(roomRepo, scheduleRepo),
		SlotUC:        slotuc.NewService(roomRepo, slotRepo),
		BookingUC:     bookingService,
		WaitlistUC:    waitlistService,
		ReservationUC: reservationService,
		WSHandler:     wsHandler,
		JWTSecret:     cfg.JWTSecret,
		CORSOrigins:   cfg.CORSAllowedOrigins,
	})
	server := httptransport.NewServer(cfg.Port, router)
	errCh := make(chan error, 1)

	go func() {
		slog.Info("http server listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
	case err := <-errCh:
		runErr = fmt.Errorf("http server failed: %w", err)
		stop()
	case err := <-subscriberErrCh:
		runErr = fmt.Errorf("realtime subscriber failed: %w", err)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Warn("graceful shutdown failed", "error", err)
	}

	done := make(chan struct{})
	go func() {
		backgroundWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-shutdownCtx.Done():
		slog.Warn("background shutdown timeout reached")
	}

	if runErr != nil {
		slog.Error("application terminated with error", "error", runErr)
	}
}
