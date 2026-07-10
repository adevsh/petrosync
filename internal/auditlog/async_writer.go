package auditlog

import (
	"context"
	"time"

	"github.com/adevsh/petrosync/internal/db"
)

type Sink interface {
	InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.InsertAuditLogRow, error)
}

type AsyncWriter struct {
	sink    Sink
	timeout time.Duration
}

func NewAsyncWriter(sink Sink) *AsyncWriter {
	return &AsyncWriter{sink: sink, timeout: 3 * time.Second}
}

func (w *AsyncWriter) Write(arg db.InsertAuditLogParams) {
	if w == nil || w.sink == nil {
		return
	}
	go func(p db.InsertAuditLogParams) {
		ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
		defer cancel()
		_, _ = w.sink.InsertAuditLog(ctx, p)
	}(arg)
}

