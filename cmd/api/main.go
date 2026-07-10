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

	"github.com/adevsh/petrosync/internal/auditlog"
	"github.com/adevsh/petrosync/internal/bot"
	"github.com/adevsh/petrosync/internal/config"
	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/handler"
	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/service"
	"github.com/adevsh/petrosync/internal/telegram"
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
	store := db.NewStore(dbPool)
	querier := store.Queries
	auditWriter := auditlog.NewAsyncWriter(querier)
	authService := service.NewAuthService(querier, cfg.JWTSecret, valkeySvc)
	workflowSvc := service.NewWorkflowService(store)
	garageStorage, err := service.NewGarageStorage(context.Background(), cfg.GarageEndpoint, cfg.GarageAccessKey, cfg.GarageSecretKey, cfg.GarageBucket)
	if err != nil {
		log.Fatalf("failed to init Garage client: %v", err)
	}
	tripPhotoSvc := service.NewTripPhotoService(querier, garageStorage)
	telegramLinkSvc := service.NewTelegramLinkService(service.NewPgxTelegramLinkStore(dbPool, querier))

	// ── Telegram bot + notification ────────────────────────────────────
	var notifSvc *service.NotificationService
	if cfg.TelegramBotToken != "" {
		tgClient := telegram.NewClient(cfg.TelegramBotToken)
		notifSvc = service.NewNotificationService(querier, tgClient)
		go func() {
			tgBot := bot.NewTelegramBot(tgClient, telegramLinkSvc)
			_ = tgBot.Run(context.Background())
		}()
	} else {
		notifSvc = service.NewNotificationService(querier, nil)
		log.Println("TELEGRAM_BOT_TOKEN not set — notifications disabled")
	}
	_ = notifSvc

	// ── Handlers ──────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authService, cfg.JWTSecret)
	userHandler := handler.NewUserHandler(querier, valkeySvc)
	refHandler := handler.NewRefineryHandler(querier)
	vehHandler := handler.NewVehicleHandler(querier)
	drvHandler := handler.NewDriverHandler(querier)
	stnHandler := handler.NewStationHandler(querier)
	tankHandler := handler.NewStorageTankHandler(querier)
	doHandler := handler.NewDeliveryOrderHandler(querier, workflowSvc, notifSvc)
	tripHandler := handler.NewTripHandler(querier, workflowSvc, tripPhotoSvc)
	gpsHandler := handler.NewGPSHandler(querier, valkeySvc)
	qrHandler := handler.NewQRHandler(querier)
	tgLinkTokenHandler := handler.NewTelegramLinkTokenHandler(querier)
	resetPwHandler := handler.NewResetPasswordHandler(querier, nil)
	wsHub := ws.NewHub()
	wsBridgeCtx, wsBridgeCancel := context.WithCancel(context.Background())
	wsSub := valkeySvc.Client().B().Psubscribe().Pattern("ws:trip:*").Build()
	go func() {
		if err := ws.RunValkeyBridge(wsBridgeCtx, valkeySvc.Client(), wsSub, wsHub); err != nil && wsBridgeCtx.Err() == nil {
			log.Printf("valkey ws bridge stopped: %v", err)
		}
	}()

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
	auth.Use(middleware.AuditTrail(auditWriter))
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", middleware.JWTAuth(cfg.JWTSecret, querier, valkeySvc), authHandler.Logout)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/change-password", middleware.JWTAuth(cfg.JWTSecret, querier, valkeySvc), authHandler.ChangePassword)
	}

	// ── Protected API v1 routes ────────────────────────────────────────
	v1 := router.Group("/api/v1")
	v1.Use(middleware.JWTAuth(cfg.JWTSecret, querier, valkeySvc))
	v1.Use(middleware.AuditTrail(auditWriter))
	{
		mobile := v1.Group("")
		{
			mobile.POST("/qr/validate", qrHandler.Validate)
			mobile.POST("/gps/batch", gpsHandler.Batch)
			mobile.GET("/trips/active", tripHandler.ListActive)
			mobile.POST("/trips/:id/events", tripHandler.CreateEvent)
			mobile.POST("/trips/:id/photos", tripHandler.UploadPhoto)
			mobile.GET("/trips/:id/photos", tripHandler.ListPhotos)
		}

		ops := v1.Group("")
		ops.Use(middleware.DisallowDriver())
		{
			users := ops.Group("/users")
			users.Use(middleware.RequiredRole(querier, "SYSTEM_ADMIN", "COMPANY", ""))
			{
				users.GET("", userHandler.ListUsers)
				users.POST("", userHandler.CreateUser)
				users.GET("/:id", userHandler.GetUser)
				users.PATCH("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeactivateUser)

				users.GET("/:id/roles", userHandler.ListRoles)
				users.POST("/:id/roles", userHandler.GrantRole)
				users.DELETE("/:id/roles", userHandler.RevokeRole)

				users.POST("/:id/telegram/link-token", tgLinkTokenHandler.CreateLinkToken)
				users.POST("/:id/reset-password", resetPwHandler.ResetPassword)
			}

			ref := ops.Group("")
			ref.Use(middleware.RequiredRole(querier, "REFINERY_ADMIN", "", ""))
			{
				ref.GET("/refineries", refHandler.ListRefineries)
				ref.GET("/refineries/:id", refHandler.GetRefinery)
				ref.GET("/refineries/:id/facilities", middleware.RequiredRole(querier, "REFINERY_ADMIN", "REFINERY", "id"), refHandler.ListFacilitiesByRefinery)
			}

			fleet := ops.Group("")
			fleet.Use(middleware.RequiredRole(querier, "DEPOT_STAFF", "", ""))
			{
				fleet.GET("/vehicles", vehHandler.ListVehiclesByStatus)
				fleet.GET("/vehicles/:id", vehHandler.GetVehicle)
				fleet.POST("/vehicles", vehHandler.CreateVehicle)
				fleet.GET("/vehicles/:id/compartments", vehHandler.ListCompartments)
				fleet.POST("/vehicles/:id/compartments", vehHandler.CreateCompartment)

				fleet.GET("/drivers/:id", drvHandler.GetDriver)
				fleet.POST("/drivers", drvHandler.CreateDriver)
				fleet.POST("/drivers/:id/shift/start", drvHandler.StartShift)
				fleet.POST("/drivers/:id/shift/end", drvHandler.EndShift)
			}

			stations := ops.Group("")
			stations.Use(middleware.RequiredRole(querier, "STATION_MANAGER", "", ""))
			{
				stations.GET("/stations", stnHandler.ListStations)
				stations.GET("/stations/:id", middleware.RequiredRole(querier, "STATION_MANAGER", "STATION", "id"), stnHandler.GetStation)
				stations.POST("/stations", stnHandler.CreateStation)
				stations.GET("/stations/:id/tanks", middleware.RequiredRole(querier, "STATION_MANAGER", "STATION", "id"), stnHandler.ListTanks)
			}

			facilityOps := ops.Group("/facilities/:id")
			facilityOps.Use(middleware.RequiredRole(querier, "FACILITY_OPERATOR", "FACILITY", "id"))
			{
				facilityOps.GET("", refHandler.GetFacility)
				facilityOps.GET("/storage-tanks", tankHandler.ListByFacility)
				facilityOps.GET("/delivery-orders", doHandler.ListByFacility)
				facilityOps.GET("/dispatch-candidates", doHandler.ListDispatchQueue)
			}

			doRoutes := ops.Group("/delivery-orders")
			doRoutes.Use(middleware.RequiredRole(querier, "FACILITY_OPERATOR", "", ""))
			{
				doRoutes.GET("/:id", doHandler.Get)
				doRoutes.POST("", doHandler.Create)
				doRoutes.POST("/:id/approve", doHandler.Approve)
				doRoutes.POST("/:id/assign", doHandler.AssignVehicleAndDriver)
				doRoutes.POST("/:id/cancel", doHandler.Cancel)
				doRoutes.GET("/:id/items", doHandler.ListItems)
				doRoutes.POST("/:id/items", doHandler.CreateItem)
			}

			trips := ops.Group("/trips")
			trips.Use(middleware.RequiredRole(querier, "FACILITY_OPERATOR", "", ""))
			{
				trips.GET("/:id", tripHandler.Get)
				trips.GET("/:id/events", tripHandler.ListEvents)
				trips.GET("/:id/compartments", tripHandler.ListCompartmentDeliveries)
				trips.GET("/:id/seals", tripHandler.ListSeals)
				trips.GET("/:id/weight-bridge", tripHandler.ListWeightBridge)
				trips.POST("/:id/weight-bridge", tripHandler.CreateWeightBridge)
			}
		}
	}

	wsRoutes := router.Group("/ws")
	wsRoutes.Use(middleware.JWTQueryAuth(cfg.JWTSecret, querier, valkeySvc))
	wsRoutes.Use(middleware.DisallowDriver())
	{
		wsRoutes.GET("/trips/active", wsHub.HandleUpgrade)
	}

	// ── Graceful shutdown ──────────────────────────────────────────────
	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.AppPort), Handler: router}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down server...")
		wsBridgeCancel()
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
