package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/adevsh/petrosync/internal/db"
)

type fakeGPSQuerier struct {
	exists   map[uuid.UUID]bool
	inserted []db.InsertGPSEventParams
}

func (f *fakeGPSQuerier) InsertGPSEvent(ctx context.Context, arg db.InsertGPSEventParams) (db.InsertGPSEventRow, error) {
	f.inserted = append(f.inserted, arg)
	return db.InsertGPSEventRow{ID: 1}, nil
}

func (f *fakeGPSQuerier) CheckGPSEventUUIDExists(ctx context.Context, eventUuid uuid.UUID) (bool, error) {
	if f.exists == nil {
		return false, nil
	}
	return f.exists[eventUuid], nil
}

type fakeGPSPublisher struct {
	channels []string
	msgs     []string
}

func (f *fakeGPSPublisher) Publish(ctx context.Context, channel, message string) error {
	f.channels = append(f.channels, channel)
	f.msgs = append(f.msgs, message)
	return nil
}

func TestGPSHandler_Batch_NumericAndDuplicates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	u1 := uuid.New()
	u2 := uuid.New()

	q := &fakeGPSQuerier{
		exists: map[uuid.UUID]bool{
			u2: true,
		},
	}
	p := &fakeGPSPublisher{}

	h := NewGPSHandler(q, p)
	r := gin.New()
	r.POST("/gps/batch", h.Batch)

	body := `[
		{"event_uuid":"` + u1.String() + `","trip_id":10,"latitude":1.25,"longitude":2.5,"speed_kmh":0,"event_timestamp":"2026-07-10T10:00:00Z"},
		{"event_uuid":"` + u1.String() + `","trip_id":10,"latitude":1.25,"longitude":2.5,"event_timestamp":"2026-07-10T10:00:01Z"},
		{"event_uuid":"` + u2.String() + `","trip_id":10,"latitude":1.25,"longitude":2.5,"event_timestamp":"2026-07-10T10:00:02Z"}
	]`
	req := httptest.NewRequest(http.MethodPost, "/gps/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Accepted   int `json:"accepted"`
			Duplicates int `json:"duplicates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Accepted != 1 || resp.Data.Duplicates != 2 {
		t.Fatalf("unexpected counts: %#v", resp.Data)
	}

	if len(q.inserted) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(q.inserted))
	}
	ins := q.inserted[0]
	if ins.TripID != 10 || ins.EventUuid != u1 {
		t.Fatalf("unexpected insert params: %#v", ins)
	}
	if !ins.Latitude.Valid || !ins.Longitude.Valid {
		t.Fatalf("expected lat/long valid")
	}
	if ins.Latitude.Int.String() != "125" || ins.Latitude.Exp != -2 {
		t.Fatalf("unexpected latitude numeric: Int=%s Exp=%d", ins.Latitude.Int.String(), ins.Latitude.Exp)
	}
	if ins.Longitude.Int.String() != "25" || ins.Longitude.Exp != -1 {
		t.Fatalf("unexpected longitude numeric: Int=%s Exp=%d", ins.Longitude.Int.String(), ins.Longitude.Exp)
	}
	if !ins.SpeedKmh.Valid || ins.SpeedKmh.Int.String() != "0" || ins.SpeedKmh.Exp != 0 {
		t.Fatalf("unexpected speed numeric: %#v", ins.SpeedKmh)
	}

	if len(p.channels) != 1 || p.channels[0] != "ws:trip:10" {
		t.Fatalf("unexpected publish channels: %#v", p.channels)
	}
}
