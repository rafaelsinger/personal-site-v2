package server

import (
	"fmt"
	"net/http"
	"personal-site/internal/config"
	"strings"

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
			http.FileServer(http.Dir("./web/static/css")).ServeHTTP(w, r)
		})))
	}

	// handle static assets
	r.Route("/static", func(r chi.Router) {
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "css") && config.IsDev {
				w.Header().Set("Cache-Control", "no-store, must-revalidate")
			}
			http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))).ServeHTTP(w, r)
		})
	})

	// protected routes
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(config.TokenAuth))
		r.Use(CustomAuthenticator(config.TokenAuth))

		// TODO: rework API to use the same name, differentiate through HTTP verb
		r.Get("/admin", GetAdminPage)
		r.Get("/post", GetNewPost)
		r.Post("/post", HandleCreatePost)
		r.Delete("/post/{postID}", HandleDeletePost)
		r.Patch("/post/{postID}", HandleEditPost)

		r.Post("/markdown", HandleUploadMarkdown)
	})

	// public routes
	r.Group(func(r chi.Router) {
		r.Get("/", GetHomePage)
		r.Get("/login", GetLoginPage)
		r.Get("/projects", GetProjectsPage)
		r.Route("/blog", func(r chi.Router) {
			// TODO: add pagination (eventually)
			r.Get("/", GetAllPosts)
			r.With(PostCtx).Get("/{postSlug:[a-z-]+}", GetPost)
			r.With(PostCtx).Get("/{postSlug:[a-z-]+}/edit", EditPost)
		})
		r.Post("/login", HandleLogin)
	})

	r.NotFound(HandleNotFound)

	err := http.ListenAndServe(config.Port, handler)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server running at http://%s%s\n", config.Addr, config.Port)
}
