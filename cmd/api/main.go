// Package main is the HTTP API entrypoint for PetroSync.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adevsh/petrosync/internal/config"
	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/handler"
	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/service"
	"github.com/adevsh/petrosync/internal/ws"
)

func main() {
	// ── Configuration ──────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// ── Database pool ──────────────────────────────────────────────────
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to PostgreSQL")

	// ── Valkey ─────────────────────────────────────────────────────────
	valkeySvc, err := service.NewValkeyService(context.Background(), cfg.ValkeyAddr)
	if err != nil {
		log.Fatalf("failed to connect to Valkey: %v", err)
	}
	defer valkeySvc.Close()
	log.Println("connected to Valkey")

	// ── Services ───────────────────────────────────────────────────────
	querier := db.New(dbPool)
	authService := service.NewAuthService(querier, cfg.JWTSecret, valkeySvc)

	// ── Handlers ──────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authService, cfg.JWTSecret)
	refHandler := handler.NewRefineryHandler(querier)
	vehHandler := handler.NewVehicleHandler(querier)
	drvHandler := handler.NewDriverHandler(querier)
	stnHandler := handler.NewStationHandler(querier)
	tankHandler := handler.NewStorageTankHandler(querier)
	doHandler := handler.NewDeliveryOrderHandler(querier)
	tripHandler := handler.NewTripHandler(querier)
	gpsHandler := handler.NewGPSHandler(querier, valkeySvc)
	qrHandler := handler.NewQRHandler(querier)
	wsHub := ws.NewHub()

	// ── Router ─────────────────────────────────────────────────────────
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	// ── Health check ───────────────────────────────────────────────────
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339)})
	})

	// ── Auth routes ────────────────────────────────────────────────────
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", middleware.JWTAuth(cfg.JWTSecret), authHandler.Logout)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/change-password", middleware.JWTAuth(cfg.JWTSecret), authHandler.ChangePassword)
	}

	// ── Protected API v1 routes ────────────────────────────────────────
	v1 := router.Group("/api/v1")
	v1.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		// Read-only reference data
		v1.GET("/refineries", refHandler.ListRefineries)
		v1.GET("/refineries/:id", refHandler.GetRefinery)
		v1.GET("/refineries/:id/facilities", refHandler.ListFacilitiesByRefinery)
		v1.GET("/facilities/:id", refHandler.GetFacility)

		// Vehicles
		v1.GET("/vehicles", vehHandler.ListVehiclesByStatus)
		v1.GET("/vehicles/:id", vehHandler.GetVehicle)
		v1.POST("/vehicles", vehHandler.CreateVehicle)
		v1.GET("/vehicles/:id/compartments", vehHandler.ListCompartments)
		v1.POST("/vehicles/:id/compartments", vehHandler.CreateCompartment)

		// Drivers
		v1.GET("/drivers/:id", drvHandler.GetDriver)
		v1.POST("/drivers", drvHandler.CreateDriver)
		v1.POST("/drivers/:id/shift/start", drvHandler.StartShift)
		v1.POST("/drivers/:id/shift/end", drvHandler.EndShift)

		// Stations
		v1.GET("/stations", stnHandler.ListStations)
		v1.GET("/stations/:id", stnHandler.GetStation)
		v1.POST("/stations", stnHandler.CreateStation)
		v1.GET("/stations/:id/tanks", stnHandler.ListTanks)

		// Storage tanks
		v1.GET("/facilities/:id/storage-tanks", tankHandler.ListByFacility)

		// Delivery orders
		v1.GET("/facilities/:id/delivery-orders", doHandler.ListByFacility)
		v1.GET("/facilities/:id/dispatch-candidates", doHandler.ListDispatchQueue)
		v1.GET("/delivery-orders/:id", doHandler.Get)
		v1.POST("/delivery-orders", doHandler.Create)
		v1.POST("/delivery-orders/:id/approve", doHandler.Approve)
		v1.POST("/delivery-orders/:id/assign", doHandler.AssignVehicleAndDriver)
		v1.POST("/delivery-orders/:id/cancel", doHandler.Cancel)
		v1.GET("/delivery-orders/:id/items", doHandler.ListItems)
		v1.POST("/delivery-orders/:id/items", doHandler.CreateItem)

		// Trips
		v1.GET("/trips/active", tripHandler.ListActive)
		v1.GET("/trips/:id", tripHandler.Get)
		v1.GET("/trips/:id/events", tripHandler.ListEvents)
		v1.POST("/trips/:id/events", tripHandler.CreateEvent)
		v1.GET("/trips/:id/compartments", tripHandler.ListCompartmentDeliveries)
		v1.GET("/trips/:id/seals", tripHandler.ListSeals)
		v1.GET("/trips/:id/weight-bridge", tripHandler.ListWeightBridge)
		v1.POST("/trips/:id/weight-bridge", tripHandler.CreateWeightBridge)

		// QR validation (Android pre-validation)
		v1.POST("/qr/validate", qrHandler.Validate)

		// GPS batch (high-frequency mobile endpoint)
		v1.POST("/gps/batch", gpsHandler.Batch)
	}

	// ── WebSocket (not behind JWT — uses token query param or cookie) ───
	router.GET("/ws/trips/active", wsHub.HandleUpgrade)

	// ── Graceful shutdown ──────────────────────────────────────────────
	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.AppPort), Handler: router}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("server forced to shutdown: %v", err)
		}
	}()

	log.Printf("PetroSync API starting on :%d (%s mode)", cfg.AppPort, cfg.AppEnv)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
	log.Println("server stopped")
}
