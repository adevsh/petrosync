package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adevsh/petrosync/internal/db"
)

var (
	ErrInvalidOrExpiredLinkToken = errors.New("invalid or expired link token")
	ErrUserAlreadyLinked         = errors.New("user already linked")
	ErrTelegramAlreadyLinked     = errors.New("telegram account already linked")
)

type TelegramLinkQuerier interface {
	GetValidTelegramLinkToken(ctx context.Context, token string) (db.GetValidTelegramLinkTokenRow, error)
	GetUserByTelegramID(ctx context.Context, telegramUserID pgtype.Int8) (db.GetUserByTelegramIDRow, error)
	LinkTelegramAccount(ctx context.Context, arg db.LinkTelegramAccountParams) error
	UseTelegramLinkToken(ctx context.Context, token string) (db.TelegramLinkToken, error)
}

type TelegramLinkStore interface {
	ExecTx(ctx context.Context, fn func(q TelegramLinkQuerier) error) error
}

type PgxTelegramLinkStore struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewPgxTelegramLinkStore(pool *pgxpool.Pool, q *db.Queries) *PgxTelegramLinkStore {
	return &PgxTelegramLinkStore{pool: pool, q: q}
}

func (s *PgxTelegramLinkStore) ExecTx(ctx context.Context, fn func(q TelegramLinkQuerier) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	q := s.q.WithTx(tx)
	if err := fn(q); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

type TelegramLinkService struct {
	store TelegramLinkStore
}

func NewTelegramLinkService(store TelegramLinkStore) *TelegramLinkService {
	return &TelegramLinkService{store: store}
}

func (s *TelegramLinkService) LinkByToken(ctx context.Context, telegramUserID int64, token string) (db.GetValidTelegramLinkTokenRow, error) {
	var out db.GetValidTelegramLinkTokenRow
	err := s.store.ExecTx(ctx, func(q TelegramLinkQuerier) error {
		row, err := q.GetValidTelegramLinkToken(ctx, token)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidOrExpiredLinkToken
		}
		if err != nil {
			return err
		}
		if row.TelegramUserID.Valid {
			return ErrUserAlreadyLinked
		}

		_, err = q.GetUserByTelegramID(ctx, pgtype.Int8{Int64: telegramUserID, Valid: true})
		if err == nil {
			return ErrTelegramAlreadyLinked
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		if _, err := q.UseTelegramLinkToken(ctx, token); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidOrExpiredLinkToken
			}
			return err
		}

		if err := q.LinkTelegramAccount(ctx, db.LinkTelegramAccountParams{
			ID:             row.UserID,
			TelegramUserID: pgtype.Int8{Int64: telegramUserID, Valid: true},
		}); err != nil {
			return err
		}

		out = row
		return nil
	})
	return out, err
}
