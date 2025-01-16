package server

import (
	"fmt"
	"net/http"
	"personal-site/internal/config"

	"github.com/aarol/reload"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
)

func Start() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)    // log start and end of each request
	r.Use(middleware.RequestID) // add unique id to each request context
	r.Use(middleware.Recoverer) // recover and log from panic, return 500
	r.Use(middleware.RealIP)    // add request RemoteAddr to X-Real-IP

	var handler http.Handler = r

	if config.IsDev {
		// list of directories to recursively watch
		reloader := reload.New("web/static/html/", "web/static/css/", "web/static/assets/")
		handler = reloader.Handle(handler)
		r.Handle("/css/*", http.StripPrefix("/css/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store, must-revalidate")
			http.FileServer(http.Dir("./css")).ServeHTTP(w, r)
		})))
	}

	// handle static assets
	r.Route("/static", func(r chi.Router) {
		r.Get("/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))).ServeHTTP)
	})

	// protected routes
	// TODO: proper 404 page
	r.Group(func(r chi.Router) {
		// TODO: proper Forbidden page when receiving 401
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
			// TODO: add pagination (eventually)
			// r.Get("/", GetAllPosts)
			r.With(PostCtx).Get("/{postID}", GetPost)
			r.With(PostCtx).Get("/{postSlug:[a-z-]+}", GetPost)
		})
		r.Post("/login", HandleLogin)
	})

	err := http.ListenAndServe(config.Port, handler)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server running at http://%s%s\n", config.Addr, config.Port)
}
