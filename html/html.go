package html

import (
	"embed"
	"html/template"
	"io"
	"os"
)

//go:embed *
var files embed.FS
var is_dev bool

func init() {
	is_dev = os.Getenv("GO_ENV") == "development"
}

func parse(file string) *template.Template {
	if is_dev {
		// dynamically read from files for dynamic template parsing
		return template.Must(
			template.New("layout.html").ParseFiles("layout.html", file))
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
