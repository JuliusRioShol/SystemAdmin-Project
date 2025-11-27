package main

import (
	"fmt"
	"log"
	"net/http"

	"discussionboard/internal/data"

	"golang.org/x/crypto/bcrypt"
)

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		app.render(w, "login", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := data.GetUserByEmail(app.db, email)
	if err != nil || bcrypt.CompareHashAndPassword(user.Password, []byte(password)) != nil {
		app.render(w, "login", map[string]string{"Error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		app.render(w, "login", map[string]string{"Error": "Account not activated.  Check console for activation token."})
		return
	}

	// Create session
	token, err := data.CreateAuthToken(app.db, user.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	app.sessions.Set(token, user.ID)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		app.render(w, "register", nil)
		return
	}

	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate input
	if firstName == "" || lastName == "" || email == "" || password == "" {
		app.render(w, "register", map[string]string{"Error": "All fields are required"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		app.render(w, "register", map[string]string{"Error": "Failed to process password"})
		return
	}

	// Create user
	user := &data.User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Password:  hashedPassword,
		IsActive:  false,
	}

	userID, err := data.CreateUser(app.db, user)
	if err != nil {
		app.render(w, "register", map[string]string{"Error": "Email already exists"})
		return
	}

	// Generate activation token
	activationToken, err := data.CreateActivationToken(app.db, userID)
	if err != nil {
		log.Println("Failed to create activation token:", err)
	}

	// Print activation token to console instead of sending email
	activationURL := fmt.Sprintf("http://localhost:%s/activate?token=%s", app.config.Port, activationToken)
	log.Printf("\n=== ACTIVATION REQUIRED ===")
	log.Printf("User: %s %s (%s)", firstName, lastName, email)
	log.Printf("Activation URL: %s", activationURL)
	log.Printf("========================\n")

	app.render(w, "register", map[string]string{
		"Success": "Registration successful!  Check the console for your activation link.",
	})
}

func (app *App) activateHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Invalid activation link", http.StatusBadRequest)
		return
	}

	// Activate user
	err := data.ActivateUser(app.db, token)
	if err != nil {
		http.Error(w, "Invalid or expired activation token", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, `
		<html>
		<head><title>Account Activated</title></head>
		<body style="font-family: Arial; text-align: center; padding: 50px;">
			<h1 style="color: green;">Account Activated Successfully!</h1>
			<p>Your account has been activated.  You can now log in and start posting.</p>
			<p><a href="/login" style="background: #333; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px;">Login Now</a></p>
		</body>
		</html>
	`)
}

func (app *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		data.DeleteToken(app.db, cookie.Value)
		app.sessions.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
