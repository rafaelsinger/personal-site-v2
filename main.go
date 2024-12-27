package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"personal-site/html"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var db *sql.DB

type Template struct {
	Content template.HTML `json:"content"`
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)    // log start and end of each request
	r.Use(middleware.RequestID) // add unique id to each request context
	r.Use(middleware.Recoverer) // recover and log from panic, return 500
	r.Use(middleware.RealIP)    // add request RemoteAddr to X-Real-IP

	// TODO: setup and connect to db

	r.Get("/", GetHomePage)
	err := http.ListenAndServe(":3000", r)
	if err != nil {
		panic(err)
	}
}

func GetHomePage(w http.ResponseWriter, r *http.Request) {
	html.Home(w)
}
