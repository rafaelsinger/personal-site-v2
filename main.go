package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"personal-site/config"
	"personal-site/html"
	"personal-site/utils/markdown"
	"regexp"
	"strconv"
	"strings"
	"time"

	"log"

	"github.com/aarol/reload"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/golang-jwt/jwt/v5"
)

var db *sql.DB

type Template struct {
	Content template.HTML `json:"content"`
}

type Post struct {
	id         int
	user_id    int
	title      string
	slug       string
	contents   string
	created_at time.Time
	updated_at time.Time
}

type key int

const (
	postKey key = iota
)

// TODO: modularize this code better instead of having everything in main

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)    // log start and end of each request
	r.Use(middleware.RequestID) // add unique id to each request context
	r.Use(middleware.Recoverer) // recover and log from panic, return 500
	r.Use(middleware.RealIP)    // add request RemoteAddr to X-Real-IP

	var handler http.Handler = r

	if config.IsDev {
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
		r.Use(jwtauth.Verifier(config.TokenAuth))
		r.Use(jwtauth.Authenticator(config.TokenAuth))

		r.Get("/admin", GetAdminPage)
		r.Get("/new-post", GetNewPost)
		r.Post("/upload-markdown", HandleUploadMarkdown)
		r.Post("/create-post", HandleCreatePost)
	})

	// public routes
	r.Group(func(r chi.Router) {
		r.Get("/", GetHomePage)
		r.Get("/login", GetLoginPage)
		r.Route("/posts", func(r chi.Router) {
			// TODO: add pagination
			// r.Get("/", GetAllPosts)
			r.With(PostCtx).Get("/{postID}", GetPost)
			r.With(PostCtx).Get("/postSlug:[a-z-]+", GetPost)
		})
		r.Post("/login", HandleLogin)
	})

	err = http.ListenAndServe(config.Port, handler)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server running at http://%s%s\n", config.Addr, config.Port)
}

// middleware to add post to context, throw 404 if not found
func PostCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var post *Post
		var err error

		if postID := chi.URLParam(r, "postID"); postID != "" {
			postIdInt, err := strconv.Atoi(postID)
			if err != nil {
				http.Error(w, "Invalid post ID", http.StatusBadRequest)
				return
			}
			post, err = dbGetPost(postIdInt)
		} else if postSlug := chi.URLParam(r, "postSlug"); postSlug != "" {
			post, err = dbGetPostBySlug(postSlug)
		} else {
			// TODO: return a 404
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		if err != nil {
			// TODO: return a 404
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), postKey, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

func GetPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	post, ok := ctx.Value(postKey).(*Post)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}
	w.Write([]byte(fmt.Sprintf("title:%s", post.title)))
}

func generateToken(user_id int) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin":   true,
		"user_id": user_id,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	s, err := t.SignedString(config.SignKey)
	if err != nil {
		return "", err
	}
	return s, nil
}

// TODO: move db functionality into db.go

func dbGetPost(post_id int) (*Post, error) {
	var post Post
	row := db.QueryRow("SELECT * FROM post WHERE id = ?", post_id)
	err := row.Scan(&post.id, &post.user_id, &post.title, &post.slug, &post.contents, &post.created_at, &post.updated_at)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func dbGetPostBySlug(slug string) (*Post, error) {
	var post Post
	row := db.QueryRow("SELECT * FROM post WHERE slug = ?", slug)
	err := row.Scan(&post.id, &post.user_id, &post.title, &post.slug, &post.contents, &post.created_at, &post.updated_at)
	if err != nil {
		return nil, err
	}
	return &post, nil
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
	var user_id int
	// TODO: move db functionality into db.go
	row := db.QueryRow("SELECT id, is_admin FROM user WHERE username = ? AND password = ?", user, pass)
	err = row.Scan(&user_id, &is_admin)
	if err == sql.ErrNoRows || !is_admin {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	jwt, err := generateToken(user_id)
	if err != nil {
		w.Header().Set("Error", err.Error()) //TODO: better logging of errors in response body instead of header
		http.Error(w, "Error generating JWT", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    jwt,
		HttpOnly: true,
		Secure:   !config.IsDev,
		Path:     "/",
	})

	w.Header().Set("HX-Redirect", "/admin")
	w.WriteHeader(http.StatusOK)
}

func HandleUploadMarkdown(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	var buf bytes.Buffer
	file, header, err := r.FormFile("markdown")
	if err != nil {
		http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
	}
	defer file.Close()
	io.Copy(&buf, file)
	contents := buf.String()
	title := formatTitle(header.Filename)
	slug := titleToSlug(title)
	mk, err := markdown.ParseMD(contents)
	if err != nil {
		http.Error(w, "Error parsing markdown", http.StatusBadRequest)
	}
	html := fmt.Sprintf(`
		<div class="raw-container">
            <h2 id="raw-post-title">Raw</h2>
            <textarea class="raw-post" oninput={previewPostBody(this.value)} name="post-content" form="create-post-form">%s</textarea>
            <form class="upload-markdown-container" enctype="multipart/form-data" hx-post="/upload-markdown" hx-target=".raw-post" hx-swap="innerHTML">
                <input type="file" name="markdown">
                <input type="submit" value="Upload Markdown"></button>
            </form>
            <label for="post-title">Title</label>
            <input type="text" name="post-title" value="%s" oninput={previewPostTitle(this.value)} form="create-post-form">
            <label for="post-slug">Slug</label>
            <input type="text" name="post-slug" value="%s" form="create-post-form">
            <form class="create-post-container" id="create-post-form" method="post" action="/create-post">
                <button type="submit">Create Post</button>
            </form>
        </div>
		<div class="preview-container">
            <h2 id="preview-post-title">Preview</h2>
            <h3 class="preview-title">%s</h3>
            <div class="preview-post">%s</div>
        </div>
	`, mk, title, slug, title, mk)
	w.Write([]byte(html))
}

func formatTitle(filename string) string {
	return filename[:len(filename)-3]
}

func titleToSlug(title string) string {
	titleLower := strings.ToLower(title)
	// remove all punctuation
	reg, _ := regexp.Compile("[^a-zA-Z0-9 ]+")
	cleansedTitle := reg.ReplaceAllString(titleLower, "")
	slugArray := strings.Split(cleansedTitle, " ")
	slug := strings.Join(slugArray, "-")
	return slug
}

func HandleCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	cookie, err := r.Cookie("jwt")
	if err != nil {
		http.Error(w, "Error parsing JWT", http.StatusInternalServerError)
	}
	tokenString := cookie.Value
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.SignKey), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Could not verify identity from JWT", http.StatusBadRequest)
	}
	claims := token.Claims.(jwt.MapClaims)
	user_id := int(claims["user_id"].(float64)) // user_id is a float64 in the map and not an int for some reason
	postContent := r.FormValue("post-content")
	postTitle := r.FormValue("post-title")
	postSlug := r.FormValue("post-slug")
	// TODO: make this actually upload a Post type using the driver.Value interface
	result, err := db.Exec(
		"INSERT INTO post (user_id, title, slug, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);",
		user_id, postTitle, postSlug, postContent, time.Now(), time.Now())
	if err != nil {
		http.Error(w, "Error creating post", http.StatusInternalServerError)
	}
	// TODO: add redirect to page for created post
	id, err := result.LastInsertId()
	w.Write([]byte(fmt.Sprintf("created row with id %d", id)))
}
