package data

import (
	"database/sql"
)

func InitDB(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		email VARCHAR(254) UNIQUE NOT NULL,
		password_hash BYTEA NOT NULL,
		is_active BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS tokens (
		hash BYTEA PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expiry TIMESTAMP NOT NULL,
		scope VARCHAR(50) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS messages (
		id SERIAL PRIMARY KEY,
		title VARCHAR(200) NOT NULL,
		content TEXT NOT NULL,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_tokens_user ON tokens(user_id);
	CREATE INDEX IF NOT EXISTS idx_tokens_expiry ON tokens(expiry);
	`
	_, err := db.Exec(schema)
	return err
}
