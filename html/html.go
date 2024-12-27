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
			template.New("layout.html").ParseFiles("html/layout.html", file))
	} else {
		// read from embedded file system in production
		return template.Must(
			template.New("layout.html").ParseFS(files, "layout.html", file))
	}
}

func Home(w io.Writer) error {
	return parse("html/home.html").Execute(w, "")
}
