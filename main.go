package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"personal-site/html"

	"log"
	"os"

	"github.com/aarol/reload"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

var db *sql.DB
var isDev bool
var addr string
var port string

type Template struct {
	Content template.HTML `json:"content"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	isDev = os.Getenv("GO_ENV") == "development"
	addr = os.Getenv("SERVER_ADDR")
	port = fmt.Sprintf(":%s", os.Getenv("SERVER_PORT"))

}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)    // log start and end of each request
	r.Use(middleware.RequestID) // add unique id to each request context
	r.Use(middleware.Recoverer) // recover and log from panic, return 500
	r.Use(middleware.RealIP)    // add request RemoteAddr to X-Real-IP

	var handler http.Handler = r

	if isDev {
		// list of directories to recursively watch
		reloader := reload.New("html/", "css/", "assets/")
		handler = reloader.Handle(handler)
		r.Handle("/css/*", http.StripPrefix("/css/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store, must-revalidate")
			http.FileServer(http.Dir("./css")).ServeHTTP(w, r)
		})))
	}

	db, err := Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	r.Get("/", GetHomePage)

	err = http.ListenAndServe(port, handler)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server running at http://%s%s\n", addr, port)
}

func GetHomePage(w http.ResponseWriter, r *http.Request) {
	html.Home(w)
}
