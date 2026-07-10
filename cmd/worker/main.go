// Package main is the background worker entrypoint for PetroSync.
// It runs scheduled jobs: weight bridge escalation, route deviation
// detection, license/keur expiry notifications, and maintenance tasks.
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/adevsh/petrosync/internal/config"
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

	// ── Scheduled jobs ─────────────────────────────────────────────────
	// Phase 1: placeholder — jobs registered as services are built.
	// Phase 2: weight bridge escalation, route deviation, expiry notifications.
	_ = cfg // used when jobs are wired

	// ── Start ──────────────────────────────────────────────────────────
	c.Start()
	log.Println("PetroSync worker started (Asia/Jakarta)")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down worker...")
	ctx := c.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(30 * time.Second):
	}
	log.Println("worker stopped")
}
