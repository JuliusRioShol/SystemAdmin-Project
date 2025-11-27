package data

import (
	"database/sql"
)

func CreateMessage(db *sql.DB, message *Message) error {
	query := `
		INSERT INTO messages (title, content, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := db.QueryRow(
		query,
		message.Title,
		message.Content,
		message.UserID,
	).Scan(&message.ID, &message.CreatedAt)

	return err
}

func GetAllMessages(db *sql.DB) ([]Message, error) {
	query := `
		SELECT m.id, m.title, m.content, m.user_id, m.created_at, 
		       u.first_name, u.last_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		ORDER BY m.created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var firstName, lastName string

		err := rows.Scan(
			&m.ID,
			&m.Title,
			&m.Content,
			&m.UserID,
			&m.CreatedAt,
			&firstName,
			&lastName,
		)
		if err != nil {
			continue
		}

		m.UserName = firstName + " " + lastName
		messages = append(messages, m)
	}

	return messages, nil
}

func GetMessagesByUser(db *sql.DB, userID int64) ([]Message, error) {
	query := `
		SELECT m. id, m.title, m. content, m.user_id, m.created_at,
		       u.first_name, u.last_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		WHERE m.user_id = $1
		ORDER BY m.created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var firstName, lastName string

		err := rows.Scan(
			&m.ID,
			&m.Title,
			&m.Content,
			&m.UserID,
			&m.CreatedAt,
			&firstName,
			&lastName,
		)
		if err != nil {
			continue
		}

		m.UserName = firstName + " " + lastName
		messages = append(messages, m)
	}

	return messages, nil
}
