package sqlx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/themgmd/scenario"
	"github.com/themgmd/scenario/store/pkg"
)

// Storage .
type Storage struct {
	db *sqlx.DB
}

// NewStorage .
func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{db: db}
}

func (s *Storage) ensureTable(ctx context.Context) error {
	query := fmt.Sprintf(pkg.SqlEnsureTableQuery, pkg.SqlTableName)
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// GetSession .
func (s *Storage) GetSession(ctx context.Context, chatID, userID int64) (*scenario.Session, error) {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table: %v", err)
		return nil, err
	}

	var payload []byte
	query := fmt.Sprintf(pkg.SqlGetSessionQuery, pkg.SqlTableName)

	err = s.db.GetContext(ctx, &payload, query, chatID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get session: %v", err)
		return nil, err
	}

	var session scenario.Session
	err = json.Unmarshal(payload, &session)
	if err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal session: %v", err)
		return nil, err
	}

	return &session, nil
}

// SetSession .
func (s *Storage) SetSession(ctx context.Context, sess *scenario.Session) error {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table: %v", err)
		return err
	}

	payload, err := json.Marshal(sess.Data)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal session: %v", err)
		return err
	}

	query := fmt.Sprintf(pkg.SqlUpsertSessionQuery, pkg.SqlTableName)
	_, err = s.db.ExecContext(ctx, query, sess.ChatID, sess.UserID, payload, sess.Scene, sess.Step)
	if err != nil {
		slog.ErrorContext(ctx, "failed to upsert session: %v", err)
		return err
	}

	return nil
}
