package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log"
	"net/http"

	"discussionboard/internal/config"
	"discussionboard/internal/data"

	_ "github.com/lib/pq"
)

//go:embed templates/*
var templateFS embed.FS

// App holds application dependencies
type App struct {
	config    config.Config
	db        *sql.DB
	templates map[string]*template.Template
	sessions  *data.SessionManager
}

func main() {
	cfg := config.Load()

	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize database schema
	if err := data.InitDB(db); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	app := &App{
		config:   cfg,
		db:       db,
		sessions: data.NewSessionManager(),
	}

	// Load templates
	app.loadTemplates()

	// Setup routes
	app.setupRoutes()

	log.Printf("Server starting on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

func (app *App) loadTemplates() {
	app.templates = make(map[string]*template.Template)
	templateNames := []string{"home", "login", "register", "post"}

	for _, name := range templateNames {
		tmpl, err := template.ParseFS(templateFS, "templates/base.html", "templates/"+name+".html")
		if err != nil {
			log.Fatal("Failed to parse template:", err)
		}
		app.templates[name] = tmpl
	}
}

func (app *App) setupRoutes() {
	http.HandleFunc("/", app.requireTemplate(app.homeHandler))
	http.HandleFunc("/login", app.requireTemplate(app.loginHandler))
	http.HandleFunc("/register", app.requireTemplate(app.registerHandler))
	http.HandleFunc("/logout", app.logoutHandler)
	http.HandleFunc("/activate", app.activateHandler)
	http.HandleFunc("/post", app.requireAuth(app.postMessageHandler))
}
