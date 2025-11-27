package data

import (
	"database/sql"
	"errors"
)

var ErrUserNotFound = errors.New("user not found")
var ErrEmailExists = errors.New("email already exists")

func CreateUser(db *sql.DB, user *User) (int64, error) {
	query := `
		INSERT INTO users (first_name, last_name, email, password_hash, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	var userID int64
	err := db.QueryRow(
		query,
		user.FirstName,
		user.LastName,
		user.Email,
		user.Password,
		user.IsActive,
	).Scan(&userID, &user.CreatedAt)

	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			return 0, ErrEmailExists
		}
		return 0, err
	}

	user.ID = userID
	return userID, nil
}

func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, first_name, last_name, email, password_hash, is_active, created_at
		FROM users WHERE email = $1
	`

	err := db.QueryRow(query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func GetUserByID(db *sql.DB, id int64) (*User, error) {
	user := &User{}
	query := `
		SELECT id, first_name, last_name, email, password_hash, is_active, created_at
		FROM users WHERE id = $1
	`

	err := db.QueryRow(query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func ActivateUser(db *sql.DB, activationToken string) error {
	// First, get user ID from token
	userID, err := GetUserIDFromToken(db, activationToken)
	if err != nil {
		return err
	}

	// Activate the user
	_, err = db.Exec("UPDATE users SET is_active = true WHERE id = $1", userID)
	if err != nil {
		return err
	}

	// Delete the activation token
	return DeleteToken(db, activationToken)
}
