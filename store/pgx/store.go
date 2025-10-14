package pgx

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/themgmd/scenario"
	"github.com/themgmd/scenario/store/pkg"
	"log/slog"
)

// Executor .
type Executor interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// Storage .
type Storage struct {
	executor Executor
}

// NewStorage .
func NewStorage(executor Executor) *Storage {
	return &Storage{executor: executor}
}

func (s *Storage) ensureTable(ctx context.Context) error {
	query := fmt.Sprintf(pkg.SqlEnsureTableQuery, pkg.SqlTableName)
	_, err := s.executor.Exec(ctx, query)
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

	err = pgxscan.Get(ctx, s.executor, &payload, query, chatID, userID)
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
	_, err = s.executor.Exec(ctx, query, chatID, userID, payload)
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
	_, err = s.executor.Exec(ctx, query, chatID, userID, name)
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
	err = pgxscan.Get(ctx, s.executor, &scene, query, chatID, userID)
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
	_, err = s.executor.Exec(ctx, query, chatID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to remove scene: %v", err)
		return
	}

	return
}
