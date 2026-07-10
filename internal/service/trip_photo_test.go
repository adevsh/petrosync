package service

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type fakeObjectStorage struct {
	putKey         string
	putContentType string
	putSize        int64
	putCalled      bool

	presignKeys []string
}

func (s *fakeObjectStorage) PutObject(ctx context.Context, key string, body io.Reader, contentType string, sizeBytes int64) error {
	s.putKey = key
	s.putContentType = contentType
	s.putSize = sizeBytes
	s.putCalled = true
	return nil
}

func (s *fakeObjectStorage) PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error) {
	s.presignKeys = append(s.presignKeys, key)
	return "https://example.invalid/" + key, nil
}

type fakeTripPhotoQuerier struct {
	createParams db.CreateTripPhotoParams
	photos       []db.TripPhoto
}

func (q *fakeTripPhotoQuerier) CreateTripPhoto(ctx context.Context, arg db.CreateTripPhotoParams) (db.TripPhoto, error) {
	q.createParams = arg
	return db.TripPhoto{ID: 14, TripID: arg.TripID, GarageObjectKey: arg.GarageObjectKey}, nil
}

func (q *fakeTripPhotoQuerier) GetTrip(ctx context.Context, id int64) (db.GetTripRow, error) {
	return db.GetTripRow{ID: id}, nil
}

func (q *fakeTripPhotoQuerier) ListPhotosByTrip(ctx context.Context, tripID int64) ([]db.TripPhoto, error) {
	return q.photos, nil
}

func TestTripPhotoService_UploadTripPhoto_UsesObjectKeyConventionAndStoresKey(t *testing.T) {
	q := &fakeTripPhotoQuerier{}
	store := &fakeObjectStorage{}
	svc := NewTripPhotoService(q, store)
	svc.uuidFn = func() string { return "fixed-uuid" }

	now := time.Now()
	r := bytes.NewReader([]byte("jpegbytes"))
	photo, err := svc.UploadTripPhoto(
		context.Background(),
		42,
		99,
		db.PhotoEventTWEIGHTBRIDGETARE,
		pgtype.Int8{},
		now,
		pgtype.Text{},
		r,
		1234,
		"image/jpeg",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.putCalled {
		t.Fatalf("expected PutObject to be called")
	}
	if store.putKey != "trips/42/photos/WEIGHT_BRIDGE_TARE/fixed-uuid.jpg" {
		t.Fatalf("unexpected object key: %s", store.putKey)
	}
	if q.createParams.GarageObjectKey != store.putKey {
		t.Fatalf("expected garage object key to match upload key")
	}
	if photo.ID != 14 {
		t.Fatalf("unexpected photo id: %d", photo.ID)
	}
}

func TestTripPhotoService_ListTripPhotosWithURLs_AddsPresignedURLs(t *testing.T) {
	q := &fakeTripPhotoQuerier{
		photos: []db.TripPhoto{
			{ID: 1, TripID: 42, GarageObjectKey: "trips/42/photos/WEIGHT_BRIDGE_TARE/a.jpg"},
			{ID: 2, TripID: 42, GarageObjectKey: "trips/42/photos/WEIGHT_BRIDGE_GROSS/b.jpg"},
		},
	}
	store := &fakeObjectStorage{}
	svc := NewTripPhotoService(q, store)

	out, err := svc.ListTripPhotosWithURLs(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(out))
	}
	if out[0].PresignedGetURL == "" || out[1].PresignedGetURL == "" {
		t.Fatalf("expected presigned urls to be set")
	}
	if len(store.presignKeys) != 2 {
		t.Fatalf("expected 2 presign calls, got %d", len(store.presignKeys))
	}
}
