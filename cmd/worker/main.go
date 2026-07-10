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
	escalationSvc := service.NewWeightBridgeEscalationService(q, notifSvc)
	_, _ = c.AddFunc("*/30 * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		n, err := escalationSvc.Run(ctx)
		if err != nil {
			log.Printf("weight bridge escalation job failed: %v", err)
			return
		}
		if n > 0 {
			log.Printf("weight bridge escalation job: escalated %d readings", n)
		}
	})

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
