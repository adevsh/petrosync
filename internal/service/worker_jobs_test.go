package service

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"
)

func TestNewLoggedJob_RecoversPanicAndLogsStack(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	job := NewLoggedJob(logger, "panic-test", time.Second, func(ctx context.Context) error {
		panic("boom")
	})

	job()

	out := buf.String()
	if !strings.Contains(out, "job panic-test started") {
		t.Fatalf("expected start log, got %q", out)
	}
	if !strings.Contains(out, "job panic-test panic: boom") {
		t.Fatalf("expected panic log, got %q", out)
	}
}

func TestNewLoggedJob_LogsCompletion(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	job := NewLoggedJob(logger, "success-test", time.Second, func(ctx context.Context) error {
		return nil
	})

	job()

	out := buf.String()
	if !strings.Contains(out, "job success-test completed") {
		t.Fatalf("expected completion log, got %q", out)
	}
}
