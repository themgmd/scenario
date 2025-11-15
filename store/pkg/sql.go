package pkg

const (
	SqlTableName = "telegram_scene_sessions"
)

const (
	SqlEnsureTableQuery = `CREATE TABLE IF NOT EXISTS %s (
		chat_id BIGINT NOT NULL,
		user_id BIGINT NOT NULL,
		scene TEXT,
		step INTEGER NOT NULL DEFAULT -1,
		data JSONB NOT NULL DEFAULT '{}'::jsonb,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		PRIMARY KEY (chat_id, user_id)
	)`

	SqlUpsertSessionQuery = `INSERT INTO %s (chat_id, user_id, data, scene, step, updated_at) VALUES ($1, $2, $3, $4, $5, NOW()) ON CONFLICT (chat_id, user_id) DO UPDATE SET session = excluded.session, scene = excluded.scene, step = excluede.step, updated_at = NOW()`

	SqlGetSessionQuery = `SELECT * FROM %s WHERE chat_id=$1 AND user_id=$2`
)
