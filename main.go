// Simple Web app for Testing
package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
)

var db *pgx.Conn
var templates = template.Must(template.ParseGlob("templates/*.html"))

func main() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		dbUser, dbPass, dbHost, dbPort, dbName,
	)

	var err error
	db, err = pgx.Connect(context.Background(), connStr)
	if err != nil {
		panic(err)
	}

	fmt.Println("Connected to PostgreSQL inside Docker :)")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)

	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value != "authenticated" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	templates.ExecuteTemplate(w, "home.html", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		templates.ExecuteTemplate(w, "login.html", nil)
		return
	}

	r.ParseForm()
	user := r.FormValue("username")
	pass := r.FormValue("password")

	if user == "admin" && pass == "12345" {
		http.SetCookie(w, &http.Cookie{
			Name:  "session",
			Value: "authenticated",
			Path:  "/",
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		w.Write([]byte("Invalid login"))
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
