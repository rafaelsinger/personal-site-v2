package html

import (
	"embed"
	"html/template"
	"io"
	"personal-site/internal/config"
	"personal-site/internal/db"
	"personal-site/internal/types"
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

func Home(w io.Writer, posts []*db.Post) error {
	return parse("home.html").Execute(w, posts)
}

func Error(w io.Writer, err types.StatusError) error {
	return parse("error.html").Execute(w, err)
}

func Login(w io.Writer) error {
	return parse("login.html").Execute(w, "")
}

func Admin(w io.Writer, posts []*db.Post) error {
	return parse("admin.html").Execute(w, posts)
}

func NewPost(w io.Writer) error {
	return parse("new-post.html").Execute(w, "")
}

func Projects(w io.Writer) error {
	return parse("projects.html").Execute(w, "")
}

func Post(w io.Writer, postData *db.PostData) error {
	return parse("post.html").Execute(w, postData)
}

func Edit(w io.Writer, post *db.Post) error {
	return parse("edit.html").Execute(w, post)
}

func AllPosts(w io.Writer, blogData *db.BlogData) error {
	return parse("blog.html").Execute(w, blogData)
}
