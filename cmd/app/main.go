package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"sync"
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

	redis "github.com/redis/go-redis/v9"
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

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	pingCtx, pingCancel := context.WithTimeout(ctx, time.Second)
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		log.Printf("Redis unavailable at startup (%s): %v; realtime will run in degraded mode until reconnect", cfg.RedisAddr, err)
	} else {
		log.Printf("Redis connected: %s db=%d channel=%s", cfg.RedisAddr, cfg.RedisDB, cfg.RedisChannel)
	}
	pingCancel()
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("close redis client: %v", err)
		}
	}()

	realtimeHub := realtime.NewHub()
	realtimePublisher := realtime.NewRedisPublisher(redisClient, cfg.RedisChannel)
	realtimeSubscriber := realtime.NewRedisSubscriber(redisClient, cfg.RedisChannel, realtimeHub)
	wsHandler := realtime.NewWSHandler(realtimeHub, cfg.JWTSecret)

	var backgroundWG sync.WaitGroup
	subscriberErrCh := make(chan error, 1)
	backgroundWG.Add(1)
	go func() {
		defer backgroundWG.Done()
		if err := realtimeSubscriber.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			subscriberErrCh <- err
		}
	}()

	router := httptransport.NewRouterWithDependencies(httptransport.RouterDependencies{
		AuthUC:     authsvc.NewService(authsvc.NewHMACJWTSigner(cfg.JWTSecret), userRepo),
		RoomUC:     roomuc.NewService(roomRepo),
		ScheduleUC: scheduleuc.NewService(roomRepo, scheduleRepo),
		SlotUC:     slotuc.NewService(roomRepo, slotRepo),
		BookingUC:  bookinguc.NewService(txManager, bookingRepo, slotRepo, realtimePublisher),
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
		log.Printf("graceful shutdown failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		backgroundWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-shutdownCtx.Done():
		log.Printf("background shutdown timeout reached")
	}

	if runErr != nil {
		log.Fatalf("%v", runErr)
	}
}
