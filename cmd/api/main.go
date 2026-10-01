package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/ismaelucky94/near_u_exercise/internal/clock"
	"github.com/ismaelucky94/near_u_exercise/internal/config"
	"github.com/ismaelucky94/near_u_exercise/internal/db"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories/memory"
	"github.com/ismaelucky94/near_u_exercise/internal/repositories/postgres"
	"github.com/ismaelucky94/near_u_exercise/internal/routes"
	"github.com/ismaelucky94/near_u_exercise/internal/seed"
	"github.com/ismaelucky94/near_u_exercise/internal/services"
)

func main() {
	cfg := config.Load()

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	funds, accounts := seed.Load()

	var (
		fundRepo        repositories.FundRepository
		accountRepo     repositories.AccountRepository
		orderRepo       repositories.OrderRepository
		eventRepo       repositories.EventRepository
		idempotencyRepo repositories.IdempotencyRepository
		navRepo         repositories.NAVRepository
		uow             repositories.UnitOfWork
		ready           = func() bool { return true }
	)

	if cfg.DatabaseURL != "" {
		ctx := context.Background()
		pool, err := db.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("database connection failed: %v", err)
		}
		defer pool.Close()

		if err := db.MigrateFromFile(ctx, pool, "migrations/001_init.up.sql"); err != nil {
			log.Fatalf("database migration failed: %v", err)
		}
		if err := postgres.Seed(ctx, pool, funds, accounts); err != nil {
			log.Fatalf("database seed failed: %v", err)
		}

		fundRepo = postgres.NewFundRepository(pool)
		accountRepo = postgres.NewAccountRepository(pool)
		orderRepo = postgres.NewOrderRepository(pool)
		eventRepo = postgres.NewEventRepository(pool)
		idempotencyRepo = postgres.NewIdempotencyRepository(pool)
		navRepo = postgres.NewNAVRepository(pool)
		uow = postgres.NewUnitOfWork(pool)
		ready = func() bool {
			return pool.Ping(ctx) == nil
		}
	} else {
		store := memory.NewStore(funds, accounts)
		fundRepo = memory.NewFundRepository(store)
		accountRepo = memory.NewAccountRepository(store)
		orderRepo = memory.NewOrderRepository(store)
		eventRepo = memory.NewEventRepository(store)
		idempotencyRepo = memory.NewIdempotencyRepository(store)
		navRepo = memory.NewNAVRepository(store)
		uow = memory.NewUnitOfWork(store)
	}

	accSvc := services.NewAccountService(accountRepo)
	calc := services.NewTradeDateCalculator()
	clk := clock.RealClock{}
	orderSvc := services.NewOrderService(fundRepo, accSvc, orderRepo, eventRepo, idempotencyRepo, uow, calc, clk)
	pricingSvc := services.NewPricingService(fundRepo, accSvc, orderRepo, eventRepo, navRepo, uow, clk)

	router := gin.New()
	router.Use(gin.Recovery())

	routes.Register(router, routes.Dependencies{
		Orders:   orderSvc,
		Accounts: accSvc,
		Pricing:  pricingSvc,
		Ready:    ready,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-srvErr:
		log.Fatalf("server error: %v", err)
	case sig := <-quit:
		log.Printf("received signal %s, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
		log.Println("server stopped")
	}
}
