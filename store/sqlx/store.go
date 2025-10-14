package sqlx

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/themgmd/scenario"
	"github.com/themgmd/scenario/store/pkg"
	"log/slog"
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

func (s *Storage) GetSession(ctx context.Context, chatID, userID int64) *scenario.Session {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table: %v", err)
		return nil
	}

	var payload []byte
	query := fmt.Sprintf(pkg.SqlGetSessionQuery, pkg.SqlEnsureTableQuery)

	err = s.db.GetContext(ctx, &payload, query, chatID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get session: %v", err)
		return nil
	}

	var session scenario.Session
	err = json.Unmarshal(payload, &session)
	if err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal session: %v", err)
		return nil
	}

	return &session
}

func (s *Storage) SetSession(ctx context.Context, chatID, userID int64, sess *scenario.Session) {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table: %v", err)
		return
	}

	payload, err := json.Marshal(sess)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal session: %v", err)
		return
	}

	query := fmt.Sprintf(pkg.SqlUpsertSessionQuery, pkg.SqlTableName)
	_, err = s.db.ExecContext(ctx, query, chatID, userID, payload)
	if err != nil {
		slog.ErrorContext(ctx, "failed to upsert session: %v", err)
		return
	}

	return
}

func (s *Storage) SetScene(ctx context.Context, chatID, userID int64, name string) {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table: %v", err)
		return
	}

	query := fmt.Sprintf(pkg.SqlUpsertSceneQuery, pkg.SqlTableName)
	_, err = s.db.ExecContext(ctx, query, chatID, userID, name)
	if err != nil {
		slog.ErrorContext(ctx, "failed to upsert scene: %v", err)
		return
	}

	return
}

func (s *Storage) GetScene(ctx context.Context, chatID, userID int64) string {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table: %v", err)
		return ""
	}

	var scene string
	query := fmt.Sprintf(pkg.SqlGetSceneQuery, pkg.SqlTableName)
	err = s.db.GetContext(ctx, &scene, query, chatID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get scene: %v", err)
		return ""
	}

	return scene
}

func (s *Storage) RemoveScene(ctx context.Context, chatID, userID int64) {
	err := s.ensureTable(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to ensure table: %v", err)
		return
	}

	query := fmt.Sprintf(pkg.SqlRemoveSceneQuery, pkg.SqlTableName)
	_, err = s.db.ExecContext(ctx, query, chatID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to remove scene: %v", err)
		return
	}

	return
}
