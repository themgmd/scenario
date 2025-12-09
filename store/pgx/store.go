package pgx

import (
	"context"
	"errors"
	"fmt"
	"sync"

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
	executor     Executor
	tableEnsured sync.Once
}

// NewStorage .
func NewStorage(executor Executor) (*Storage, error) {
	storage := &Storage{executor: executor}
	err := storage.ensureTable(context.Background())
	if err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *Storage) ensureTable(ctx context.Context) error {
	var err error
	s.tableEnsured.Do(func() {
		query := fmt.Sprintf(pkg.SqlEnsureTableQuery, pkg.SqlTableName)
		_, err = s.executor.Exec(ctx, query)
	})
	return err
}

// GetSession .
func (s *Storage) GetSession(ctx context.Context, chatID, userID int64) (*scenario.SessionBase, error) {
	query := fmt.Sprintf(pkg.SqlGetSessionQuery, pkg.SqlTableName)

	var session scenario.SessionBase
	err := pgxscan.Get(ctx, s.executor, &session, query, chatID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, scenario.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %v", err)
	}

	return &session, nil
}

// SetSession .
func (s *Storage) SetSession(ctx context.Context, sess *scenario.SessionBase) error {
	payload := sess.Data
	if payload == nil {
		payload = []byte("{}")
	}

	query := fmt.Sprintf(pkg.SqlUpsertSessionQuery, pkg.SqlTableName)
	_, err := s.executor.Exec(ctx, query, sess.ChatID, sess.UserID, payload, sess.Scene, sess.Step)
	if err != nil {
		return fmt.Errorf("failed to upsert session: %v", err)
	}

	return nil
}
