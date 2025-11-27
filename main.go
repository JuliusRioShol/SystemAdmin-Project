package main

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*
var templateFS embed.FS

// Config holds database configuration
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

// App holds application dependencies
type App struct {
	db        *sql.DB
	templates map[string]*template.Template
	sessions  *SessionManager
	mailer    *Mailer
}

// SessionManager handles user sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]int64 // token -> userID
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]int64),
	}
}

func (sm *SessionManager) Set(token string, userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[token] = userID
}

func (sm *SessionManager) Get(token string) (int64, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	userID, exists := sm.sessions[token]
	return userID, exists
}

func (sm *SessionManager) Delete(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, token)
}

// User represents a user
type User struct {
	ID        int64
	FirstName string
	LastName  string
	Email     string
	Password  []byte
	IsActive  bool
	CreatedAt time.Time
}

// Message represents a discussion board message
type Message struct {
	ID        int64
	Title     string
	Content   string
	UserID    int64
	UserName  string
	CreatedAt time.Time
}

// Token represents an authentication token
type Token struct {
	Plaintext string
	Hash      []byte
	UserID    int64
	Expiry    time.Time
	Scope     string
}

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

func main() {
	cfg := Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "discussiondb"),
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatal("Database not available:", err)
	}

	// Initialize database schema
	if err := initDB(db); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	app := &App{
		db:       db,
		sessions: NewSessionManager(),
		mailer:   NewMailer(), // You can configure this with your SMTP settings
	}

	// Load templates
	app.templates = make(map[string]*template.Template)
	templateNames := []string{"home", "login", "register", "post"}
	for _, name := range templateNames {
		tmpl, err := template.ParseFS(templateFS, "templates/base.html", "templates/"+name+".html")
		if err != nil {
			log.Fatal("Failed to parse template:", err)
		}
		app.templates[name] = tmpl
	}

	// Routes
	http.HandleFunc("/", app.homeHandler)
	http.HandleFunc("/login", app.loginHandler)
	http.HandleFunc("/register", app.registerHandler)
	http.HandleFunc("/logout", app.logoutHandler)
	http.HandleFunc("/activate", app.activateHandler)
	http.HandleFunc("/post", app.postMessageHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB(db *sql.DB) error {
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
	`
	_, err := db.Exec(schema)
	return err
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	user := app.getCurrentUser(r)

	// Get all messages
	rows, err := app.db.Query(`
		SELECT m.id, m.title, m.content, m.user_id, m.created_at, u.first_name, u.last_name
		FROM messages m
		JOIN users u ON m.user_id = u.id
		ORDER BY m.created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var firstName, lastName string
		err := rows.Scan(&m.ID, &m.Title, &m.Content, &m.UserID, &m.CreatedAt, &firstName, &lastName)
		if err != nil {
			continue
		}
		m.UserName = firstName + " " + lastName
		messages = append(messages, m)
	}

	data := struct {
		User     *User
		Messages []Message
	}{
		User:     user,
		Messages: messages,
	}

	app.templates["home"].Execute(w, data)
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		app.templates["login"].Execute(w, nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Get user by email
	var user User
	err := app.db.QueryRow(`
		SELECT id, first_name, last_name, email, password_hash, is_active
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password, &user.IsActive)

	if err != nil || bcrypt.CompareHashAndPassword(user.Password, []byte(password)) != nil {
		app.templates["login"].Execute(w, map[string]string{"Error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		app.templates["login"].Execute(w, map[string]string{"Error": "Account not activated. Check your email."})
		return
	}

	// Create session token
	token := generateToken()
	tokenHash := sha256.Sum256([]byte(token))

	// Store in database
	_, err = app.db.Exec(`
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)
	`, tokenHash[:], user.ID, time.Now().Add(24*time.Hour), ScopeAuthentication)

	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Store in memory
	app.sessions.Set(token, user.ID)

	// Set cookie
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
		app.templates["register"].Execute(w, nil)
		return
	}

	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		app.templates["register"].Execute(w, map[string]string{"Error": "Failed to process password"})
		return
	}

	// Insert user
	var userID int64
	err = app.db.QueryRow(`
		INSERT INTO users (first_name, last_name, email, password_hash, is_active)
		VALUES ($1, $2, $3, $4, false)
		RETURNING id
	`, firstName, lastName, email, hashedPassword).Scan(&userID)

	if err != nil {
		app.templates["register"].Execute(w, map[string]string{"Error": "Email already exists"})
		return
	}

	// Generate activation token
	token := generateToken()
	tokenHash := sha256.Sum256([]byte(token))

	_, err = app.db.Exec(`
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)
	`, tokenHash[:], userID, time.Now().Add(72*time.Hour), ScopeActivation)

	if err != nil {
		log.Println("Failed to create activation token:", err)
	}

	// Send activation email
	if app.mailer != nil {
		activationURL := fmt.Sprintf("http://localhost:8080/activate?token=%s", token)
		app.mailer.SendActivation(email, firstName, activationURL)
	}

	app.templates["register"].Execute(w, map[string]string{
		"Success": "Registration successful! Check your email for activation link.",
	})
}

func (app *App) activateHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Invalid activation link", http.StatusBadRequest)
		return
	}

	tokenHash := sha256.Sum256([]byte(token))

	// Get user from token
	var userID int64
	err := app.db.QueryRow(`
		SELECT user_id FROM tokens
		WHERE hash = $1 AND scope = $2 AND expiry > $3
	`, tokenHash[:], ScopeActivation, time.Now()).Scan(&userID)

	if err != nil {
		http.Error(w, "Invalid or expired activation token", http.StatusBadRequest)
		return
	}

	// Activate user
	_, err = app.db.Exec("UPDATE users SET is_active = true WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Failed to activate account", http.StatusInternalServerError)
		return
	}

	// Delete activation token
	app.db.Exec("DELETE FROM tokens WHERE hash = $1", tokenHash[:])

	fmt.Fprintf(w, "<h1>Account activated successfully!</h1><p><a href='/login'>Login now</a></p>")
}

func (app *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		tokenHash := sha256.Sum256([]byte(cookie.Value))
		app.db.Exec("DELETE FROM tokens WHERE hash = $1", tokenHash[:])
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

func (app *App) postMessageHandler(w http.ResponseWriter, r *http.Request) {
	user := app.getCurrentUser(r)
	if user == nil || !user.IsActive {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		app.templates["post"].Execute(w, user)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")

	_, err := app.db.Exec(`
		INSERT INTO messages (title, content, user_id)
		VALUES ($1, $2, $3)
	`, title, content, user.ID)

	if err != nil {
		app.templates["post"].Execute(w, map[string]interface{}{
			"User":  user,
			"Error": "Failed to post message",
		})
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) getCurrentUser(r *http.Request) *User {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil
	}

	userID, exists := app.sessions.Get(cookie.Value)
	if !exists {
		// Check database
		tokenHash := sha256.Sum256([]byte(cookie.Value))
		err := app.db.QueryRow(`
			SELECT user_id FROM tokens
			WHERE hash = $1 AND scope = $2 AND expiry > $3
		`, tokenHash[:], ScopeAuthentication, time.Now()).Scan(&userID)

		if err != nil {
			return nil
		}
		app.sessions.Set(cookie.Value, userID)
	}

	var user User
	err = app.db.QueryRow(`
		SELECT id, first_name, last_name, email, is_active
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.IsActive)

	if err != nil {
		return nil
	}

	return &user
}

func generateToken() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// Mailer - simple email sender
type Mailer struct {
	// Add SMTP configuration if needed
}

func NewMailer() *Mailer {
	return &Mailer{}
}

func (m *Mailer) SendActivation(email, name, activationURL string) {
	// In production, implement actual email sending
	log.Printf("ACTIVATION EMAIL for %s (%s): %s", name, email, activationURL)
}
