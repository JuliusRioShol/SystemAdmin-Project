package main

import (
	"net/http"

	"discussionboard/internal/data"
)

func (app *App) render(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := app.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *App) getCurrentUser(r *http.Request) *data.User {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil
	}

	userID, exists := app.sessions.Get(cookie.Value)
	if !exists {
		// Check database
		userID, err = data.GetUserIDFromToken(app.db, cookie.Value)
		if err != nil {
			return nil
		}
		app.sessions.Set(cookie.Value, userID)
	}

	user, err := data.GetUserByID(app.db, userID)
	if err != nil {
		return nil
	}

	return user
}
