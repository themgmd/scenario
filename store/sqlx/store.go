package sqlx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/themgmd/scenario"
	"github.com/themgmd/scenario/store/pkg"
)

// Storage .
type Storage struct {
	db           *sqlx.DB
	tableEnsured sync.Once
}

// NewStorage .
func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{db: db}
}

func (s *Storage) ensureTable(ctx context.Context) error {
	var err error
	s.tableEnsured.Do(func() {
		query := fmt.Sprintf(pkg.SqlEnsureTableQuery, pkg.SqlTableName)
		_, err = s.db.ExecContext(ctx, query)
	})
	return err
}

// GetSession .
func (s *Storage) GetSession(ctx context.Context, chatID, userID int64) (*scenario.SessionBase, error) {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table", "error", err)
		return nil, err
	}

	query := fmt.Sprintf(pkg.SqlGetSessionQuery, pkg.SqlTableName)

	var session scenario.SessionBase
	err = s.db.GetContext(ctx, &session, query, chatID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, scenario.ErrSessionNotFound
		}
		slog.ErrorContext(ctx, "failed to get session", "error", err)
		return nil, err
	}

	return &session, nil
}

// SetSession .
func (s *Storage) SetSession(ctx context.Context, sess *scenario.SessionBase) error {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table", "error", err)
		return err
	}

	payload := sess.Data
	if payload == nil {
		payload = []byte("{}")
	}

	query := fmt.Sprintf(pkg.SqlUpsertSessionQuery, pkg.SqlTableName)
	_, err = s.db.ExecContext(ctx, query, sess.ChatID, sess.UserID, payload, sess.Scene, sess.Step)
	if err != nil {
		slog.ErrorContext(ctx, "failed to upsert session", "error", err)
		return err
	}

	return nil
}
