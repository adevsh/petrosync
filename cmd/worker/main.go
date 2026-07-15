// Package main is the background worker entrypoint for PetroSync.
// It runs scheduled jobs: weight bridge escalation, route deviation
// detection, license/keur expiry notifications, and maintenance tasks.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"

	"github.com/adevsh/petrosync/internal/bot"
	"github.com/adevsh/petrosync/internal/config"
	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/service"
	"github.com/adevsh/petrosync/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Fatalf("failed to load timezone: %v", err)
	}

	c := cron.New(cron.WithLocation(loc))

	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	// ── Scheduled jobs ─────────────────────────────────────────────────
	q := db.New(dbPool)

	var tgClient *telegram.Client
	if cfg.TelegramBotToken != "" {
		tgClient = telegram.NewClient(cfg.TelegramBotToken)
	}

	notifSvc := service.NewNotificationService(q, tgClient)
	notifications := service.NewNotificationCoordinator(q, notifSvc)
	escalationSvc := service.NewWeightBridgeEscalationService(q, notifSvc)
	expirySvc := service.NewExpiryNotificationService(q, notifications)
	routeDeviationMonitor := service.NewRouteDeviationMonitorService(q)
	routeDeviationAlerts := service.NewRouteDeviationAlertService(q, notifications)
	routeDeviationSvc := service.NewRouteDeviationWorkerService(routeDeviationMonitor, routeDeviationAlerts)
	linkCleanupSvc := service.NewTelegramLinkCleanupService(q)

	_, _ = c.AddFunc("*/30 * * * *", service.NewLoggedJob(log.Default(), "weight bridge escalation", 2*time.Minute, func(ctx context.Context) error {
		n, err := escalationSvc.Run(ctx)
		service.LogJobCount(log.Default(), "weight bridge escalation", n)
		return err
	}))
	_, _ = c.AddFunc("* * * * *", service.NewLoggedJob(log.Default(), "route deviation alert", 2*time.Minute, func(ctx context.Context) error {
		created, resolved, alerted, err := routeDeviationSvc.Run(ctx)
		service.LogJobCount(log.Default(), "route deviation detected", created)
		service.LogJobCount(log.Default(), "route deviation resolved", resolved)
		service.LogJobCount(log.Default(), "route deviation alert", alerted)
		return err
	}))
	_, _ = c.AddFunc("0 7 * * *", service.NewLoggedJob(log.Default(), "driver license expiry notification", 2*time.Minute, func(ctx context.Context) error {
		n, err := expirySvc.NotifyExpiringLicenses(ctx)
		service.LogJobCount(log.Default(), "driver license expiry notification", n)
		return err
	}))
	_, _ = c.AddFunc("0 7 * * *", service.NewLoggedJob(log.Default(), "vehicle keur expiry notification", 2*time.Minute, func(ctx context.Context) error {
		n, err := expirySvc.NotifyExpiringKeur(ctx)
		service.LogJobCount(log.Default(), "vehicle keur expiry notification", n)
		return err
	}))
	_, _ = c.AddFunc("0 2 * * *", service.NewLoggedJob(log.Default(), "telegram link token cleanup", 2*time.Minute, func(ctx context.Context) error {
		n, err := linkCleanupSvc.CleanupExpired(ctx)
		service.LogJobCount64(log.Default(), "telegram link token cleanup", n)
		return err
	}))

	botCtx, botCancel := context.WithCancel(context.Background())
	if cfg.TelegramBotToken != "" {
		linkStore := service.NewPgxTelegramLinkStore(dbPool, q)
		linkSvc := service.NewTelegramLinkService(linkStore)
		tgBot := bot.NewTelegramBot(tgClient, linkSvc)
		go func() {
			_ = tgBot.Run(botCtx)
		}()
	}

	// ── Start ──────────────────────────────────────────────────────────
	c.Start()
	log.Println("PetroSync worker started (Asia/Jakarta)")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down worker...")
	botCancel()
	ctx := c.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(30 * time.Second):
	}
	log.Println("worker stopped")
}
