package data

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"
)

func CreateAuthToken(db *sql.DB, userID int64) (string, error) {
	token := generateToken()
	tokenHash := sha256.Sum256([]byte(token))

	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)
	`

	_, err := db.Exec(
		query,
		tokenHash[:],
		userID,
		time.Now().Add(24*time.Hour),
		ScopeAuthentication,
	)

	if err != nil {
		return "", err
	}

	return token, nil
}

func CreateActivationToken(db *sql.DB, userID int64) (string, error) {
	token := generateToken()
	tokenHash := sha256.Sum256([]byte(token))

	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)
	`

	_, err := db.Exec(
		query,
		tokenHash[:],
		userID,
		time.Now().Add(72*time.Hour), // 3 days for activation
		ScopeActivation,
	)

	if err != nil {
		return "", err
	}

	return token, nil
}

func GetUserIDFromToken(db *sql.DB, token string) (int64, error) {
	tokenHash := sha256.Sum256([]byte(token))

	var userID int64
	query := `
		SELECT user_id FROM tokens
		WHERE hash = $1 AND expiry > $2
	`

	err := db.QueryRow(query, tokenHash[:], time.Now()).Scan(&userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func DeleteToken(db *sql.DB, token string) error {
	tokenHash := sha256.Sum256([]byte(token))

	_, err := db.Exec("DELETE FROM tokens WHERE hash = $1", tokenHash[:])
	return err
}

func DeleteExpiredTokens(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM tokens WHERE expiry < $1", time.Now())
	return err
}

func generateToken() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
