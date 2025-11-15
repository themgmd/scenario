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

// GetSession .
func (s *Storage) GetSession(ctx context.Context, chatID, userID int64) (*scenario.Session, error) {
	err := s.ensureTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure table: %v", err)
	}

	var payload []byte
	query := fmt.Sprintf(pkg.SqlGetSessionQuery, pkg.SqlEnsureTableQuery)

	err = pgxscan.Get(ctx, s.executor, &payload, query, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %v", err)
	}

	var session scenario.Session
	err = json.Unmarshal(payload, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %v", err)
	}

	return &session, nil
}

// SetSession .
func (s *Storage) SetSession(ctx context.Context, sess *scenario.Session) error {
	err := s.ensureTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to ensure table: %v", err)
	}

	payload, err := json.Marshal(sess.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	query := fmt.Sprintf(pkg.SqlUpsertSessionQuery, pkg.SqlTableName)
	_, err = s.executor.Exec(ctx, query, sess.ChatID, sess.UserID, payload, sess.Scene, sess.Step)
	if err != nil {
		return fmt.Errorf("failed to upsert session: %v", err)
	}

	return nil
}
