package html

import (
	"embed"
	"html/template"
	"io"
)

//go:embed *
var files embed.FS

var (
	home = parse("home.html")
	//blog = parse("blog.html")
)

func parse(file string) *template.Template {
	return template.Must(
		template.New("layout.html").ParseFS(files, "layout.html", file))
}

func Home(w io.Writer) error {
	return home.Execute(w, "")
}
