package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type TripPhotoQuerier interface {
	CreateTripPhoto(ctx context.Context, arg db.CreateTripPhotoParams) (db.TripPhoto, error)
	GetTrip(ctx context.Context, id int64) (db.GetTripRow, error)
	ListPhotosByTrip(ctx context.Context, tripID int64) ([]db.TripPhoto, error)
}

type TripPhotoWithURL struct {
	db.TripPhoto
	PresignedGetURL string `json:"presigned_get_url"`
}

type TripPhotoService struct {
	q      TripPhotoQuerier
	store  ObjectStorage
	uuidFn func() string
}

func NewTripPhotoService(q TripPhotoQuerier, store ObjectStorage) *TripPhotoService {
	return &TripPhotoService{
		q:      q,
		store:  store,
		uuidFn: func() string { return uuid.NewString() },
	}
}

func (s *TripPhotoService) UploadTripPhoto(
	ctx context.Context,
	tripID int64,
	userID int64,
	eventType db.PhotoEventT,
	compartmentID pgtype.Int8,
	takenAt time.Time,
	notes pgtype.Text,
	photo io.ReadSeeker,
	sizeBytes int64,
	contentType string,
) (db.TripPhoto, error) {
	if _, err := s.q.GetTrip(ctx, tripID); err != nil {
		return db.TripPhoto{}, err
	}

	objectKey := fmt.Sprintf("trips/%d/photos/%s/%s.jpg", tripID, eventType, s.uuidFn())
	if _, err := photo.Seek(0, io.SeekStart); err != nil {
		return db.TripPhoto{}, err
	}
	if err := s.store.PutObject(ctx, objectKey, photo, contentType, sizeBytes); err != nil {
		return db.TripPhoto{}, err
	}

	return s.q.CreateTripPhoto(ctx, db.CreateTripPhotoParams{
		TripID:          tripID,
		CompartmentID:   compartmentID,
		EventType:       eventType,
		GarageObjectKey: objectKey,
		FileSizeBytes:   pgtype.Int8{Int64: sizeBytes, Valid: true},
		MimeType:        contentType,
		UploadedBy:      userID,
		TakenAt:         pgtype.Timestamptz{Time: takenAt, Valid: true},
		Notes:           notes,
	})
}

func (s *TripPhotoService) ListTripPhotosWithURLs(ctx context.Context, tripID int64) ([]TripPhotoWithURL, error) {
	photos, err := s.q.ListPhotosByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]TripPhotoWithURL, 0, len(photos))
	for _, p := range photos {
		url, err := s.store.PresignGetObject(ctx, p.GarageObjectKey, 15*time.Minute)
		if err != nil {
			return nil, err
		}
		out = append(out, TripPhotoWithURL{TripPhoto: p, PresignedGetURL: url})
	}
	return out, nil
}

