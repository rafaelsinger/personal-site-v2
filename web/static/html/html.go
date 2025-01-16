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

type FormattedPost struct {
	Post *db.Post
	Time string
}

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
	// probably a better way to handle formatting the time but this works for now
	formattedPost := FormattedPost{
		post,
		post.CreatedAt.Format("Monday, January 2, 2006"),
	}
	return parse("post.html").Execute(w, formattedPost)
}
