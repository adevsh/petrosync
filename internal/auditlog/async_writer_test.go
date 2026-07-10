package auditlog

import (
	"context"
	"testing"
	"time"

	"github.com/adevsh/petrosync/internal/db"
)

type blockingSink struct {
	called  chan db.InsertAuditLogParams
	release chan struct{}
}

func (s *blockingSink) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.InsertAuditLogRow, error) {
	select {
	case s.called <- arg:
	default:
	}
	<-s.release
	return db.InsertAuditLogRow{}, nil
}

func TestAsyncWriter_WriteIsNonBlocking(t *testing.T) {
	sink := &blockingSink{
		called:  make(chan db.InsertAuditLogParams, 1),
		release: make(chan struct{}),
	}
	w := NewAsyncWriter(sink)

	params := db.InsertAuditLogParams{
		Action:     "X",
		EntityType: "Y",
	}

	done := make(chan struct{})
	go func() {
		w.Write(params)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("expected Write to return quickly")
	}

	select {
	case <-sink.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected InsertAuditLog to be called")
	}

	close(sink.release)
}

