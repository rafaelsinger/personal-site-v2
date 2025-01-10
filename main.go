package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"personal-site/html"
	"personal-site/utils/markdown"
	"time"

	"log"
	"os"

	"github.com/aarol/reload"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var db *sql.DB
var isDev bool
var addr string
var port string
var signKey []byte
var tokenAuth *jwtauth.JWTAuth

type Template struct {
	Content template.HTML `json:"content"`
}

// TODO: modularize this code better instead of having everything in main

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	isDev = os.Getenv("GO_ENV") == "development"
	addr = os.Getenv("SERVER_ADDR")
	port = fmt.Sprintf(":%s", os.Getenv("SERVER_PORT"))
	signKey = []byte(os.Getenv("SIGN_KEY"))
	tokenAuth = jwtauth.New("HS256", signKey, nil)
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

	// protected routes
	r.Group(func(r chi.Router) {
		// TODO: more graceful Forbidden page when receiving 401
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator(tokenAuth))

		r.Get("/admin", GetAdminPage)
		r.Get("/new-post", GetNewPost)
		r.Post("/upload-markdown", HandleUploadMarkdown)
	})

	// public routes
	r.Group(func(r chi.Router) {
		r.Get("/", GetHomePage)
		r.Get("/login", GetLoginPage)
		r.Post("/login", HandleLogin)
	})

	err = http.ListenAndServe(port, handler)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server running at http://%s%s\n", addr, port)
}

func GetHomePage(w http.ResponseWriter, r *http.Request) {
	html.Home(w)
}

func GetLoginPage(w http.ResponseWriter, r *http.Request) {
	html.Login(w)
}

func GetAdminPage(w http.ResponseWriter, r *http.Request) {
	html.Admin(w)
}

func GetNewPost(w http.ResponseWriter, r *http.Request) {
	html.NewPost(w)
}

func generateToken() (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin": true,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})
	s, err := t.SignedString(signKey)
	if err != nil {
		return "", err
	}
	return s, nil
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	// request validation
	err := r.ParseForm()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	user, pass := r.FormValue("username"), r.FormValue("password")
	if user == "" || pass == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var is_admin bool
	row := db.QueryRow("SELECT is_admin FROM user WHERE username = ? AND password = ?", user, pass)
	err = row.Scan(&is_admin)
	if err == sql.ErrNoRows || !is_admin {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	jwt, err := generateToken()
	if err != nil {
		w.Header().Set("Error", err.Error()) //TODO: better logging of errors in response body instead of header
		http.Error(w, "Error generating JWT", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    jwt,
		HttpOnly: true,
		Secure:   !isDev,
		Path:     "/",
	})

	w.Header().Set("HX-Redirect", "/admin")
	w.WriteHeader(http.StatusOK)
}

func HandleUploadMarkdown(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	var buf bytes.Buffer
	file, _, err := r.FormFile("markdown")
	if err != nil {
		http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
	}
	defer file.Close()
	io.Copy(&buf, file)
	contents := buf.String()
	mk, err := markdown.ParseMD(contents)
	if err != nil {
		http.Error(w, "Error parsing markdown", http.StatusBadRequest)
	}
	res := fmt.Sprintf(`<textarea class="post-text">%s</textarea>`, mk)
	w.Write([]byte(res))
}
