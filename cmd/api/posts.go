package main

import (
	"net/http"

	"discussionboard/internal/data"
)

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	user := app.getCurrentUser(r)

	// Get all messages (visible to everyone)
	messages, err := data.GetAllMessages(app.db)
	if err != nil {
		http.Error(w, "Failed to load messages", http.StatusInternalServerError)
		return
	}

	data := struct {
		User     *data.User
		Messages []data.Message
	}{
		User:     user,
		Messages: messages,
	}

	app.render(w, "home", data)
}

func (app *App) postMessageHandler(w http.ResponseWriter, r *http.Request) {
	user := app.getCurrentUser(r)
	if user == nil || !user.IsActive {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		app.render(w, "post", user)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")

	if title == "" || content == "" {
		app.render(w, "post", map[string]interface{}{
			"User":  user,
			"Error": "Title and content are required",
		})
		return
	}

	message := &data.Message{
		Title:   title,
		Content: content,
		UserID:  user.ID,
	}

	err := data.CreateMessage(app.db, message)
	if err != nil {
		app.render(w, "post", map[string]interface{}{
			"User":  user,
			"Error": "Failed to post message",
		})
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
