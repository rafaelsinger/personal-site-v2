package html

import (
	"embed"
	"html/template"
	"io"
	"personal-site/internal/config"
	"personal-site/internal/db"
)

//go:embed *
var files embed.FS

func parse(file string) *template.Template {
	if config.IsDev {
		// dynamically read from files for dynamic template parsing
		return template.Must(
			template.New("layout.html").ParseFiles("web/static/html/layout.html", "web/static/html/"+file))
	} else {
		// read from embedded file system in production
		return template.Must(
			template.New("layout.html").ParseFS(files, "layout.html", file))
	}
}

func Home(w io.Writer) error {
	return parse("home.html").Execute(w, "")
}

func Login(w io.Writer) error {
	return parse("login.html").Execute(w, "")
}

func Admin(w io.Writer) error {
	return parse("admin.html").Execute(w, "")
}

func NewPost(w io.Writer) error {
	return parse("new-post.html").Execute(w, "")
}

func Post(w io.Writer, post *db.Post) error {
	return parse("post.html").Execute(w, post)
}

func AllPosts(w io.Writer, posts []*db.Post) error {
	return parse("blog.html").Execute(w, posts)
}
