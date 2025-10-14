package pkg

const (
	SqlTableName = "telegram_scene_sessions"
)

const (
	SqlEnsureTableQuery = `CREATE TABLE IF NOT EXISTS %s (
		chat_id BIGINT NOT NULL,
		user_id BIGINT NOT NULL,
		scene TEXT,
		session JSONB NOT NULL DEFAULT '{}'::jsonb,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		PRIMARY KEY (chat_id, user_id)
	)`

	SqlUpsertSessionQuery = `INSERT INTO %s (chat_id, user_id, session, updated_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT (chat_id, user_id) DO UPDATE SET session = excluded.session, updated_at = NOW()`

	SqlGetSessionQuery = `SELECT session FROM %s WHERE chat_id=$1 AND user_id=$2`

	SqlUpsertSceneQuery = `INSERT INTO %s (chat_id, user_id, scene, updated_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT (chat_id, user_id) DO UPDATE SET scene = excluded.scene, updated_at = NOW()`

	SqlGetSceneQuery = `SELECT scene FROM %s WHERE chat_id=$1 AND user_id=$2`

	SqlRemoveSceneQuery = `UPDATE %s SET scene=NULL, updated_at=NOW() WHERE chat_id=$1 AND user_id=$2`
)
