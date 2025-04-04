package server

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"personal-site/internal/config"
	"personal-site/internal/db"
	"personal-site/internal/types"
	"personal-site/pkg/utils"
	"personal-site/pkg/utils/markdown"
	"personal-site/web/static/html"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

type key int

const (
	postKey key = iota
	tagsKey
)

// middleware to add post to context, throw 404 if not found
func PostCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var post *db.Post
		var tags []*db.Tag
		var err error

		if postID := chi.URLParam(r, "postID"); postID != "" {
			postIdInt, err := strconv.Atoi(postID)
			if err != nil {
				handleError(w, http.StatusBadRequest)
				return
			}
			post, err = db.GetPost(postIdInt)
			if err != nil {
				if err == sql.ErrNoRows {
					handleError(w, http.StatusNotFound)
				} else {
					handleError(w, http.StatusInternalServerError)
				}
				return
			}
		} else if postSlug := chi.URLParam(r, "postSlug"); postSlug != "" {
			post, err = db.GetPostBySlug(postSlug)
			if err != nil {
				if err == sql.ErrNoRows {
					handleError(w, http.StatusNotFound)
				} else {
					handleError(w, http.StatusInternalServerError)
				}
				return
			}
			tags, err = db.GetTags(post.Id)
		} else {
			handleError(w, http.StatusNotFound)
			return
		}

		if err != nil {
			handleError(w, http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), postKey, post)
		ctx = context.WithValue(ctx, tagsKey, tags)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetHomePage(w http.ResponseWriter, r *http.Request) {
	posts, err := db.GetAllPosts(db.WithLimit(3))
	if err != nil {
		handleError(w, http.StatusUnprocessableEntity)
		return
	}
	html.Home(w, posts)
}

func GetLoginPage(w http.ResponseWriter, r *http.Request) {
	html.Login(w)
}

func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	handleError(w, http.StatusNotFound)
}

func CustomAuthenticator(ja *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, _, err := jwtauth.FromContext(r.Context())

			if err != nil {
				handleError(w, http.StatusUnauthorized)
				return
			}

			if token == nil {
				handleError(w, http.StatusUnauthorized)
				return
			}

			// Token is authenticated, pass it through
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(hfn)
	}
}

// TODO: standardize date formatting, this is inefficient
func GetAdminPage(w http.ResponseWriter, r *http.Request) {
	posts, err := db.GetAllPosts()
	for _, post := range posts {
		post.Published = post.CreatedAt.Format("01/02/06")
	}
	if err != nil {
		handleError(w, http.StatusUnprocessableEntity)
		return
	}
	html.Admin(w, posts)
}

func GetNewPost(w http.ResponseWriter, r *http.Request) {
	html.NewPost(w)
}

func GetProjectsPage(w http.ResponseWriter, r *http.Request) {
	html.Projects(w)
}
func GetAllPosts(w http.ResponseWriter, r *http.Request) {
	var tagFilters []string
	var posts []*db.Post
	var err error
	params := r.URL.Query()
	for key, val := range params {
		if key == "q" {
			tagFilters = val
		}
	}
	if len(tagFilters) > 0 {
		posts, err = db.GetFilteredPosts(tagFilters)
	} else {
		posts, err = db.GetAllPosts()
	}
	if err != nil {
		handleError(w, http.StatusUnprocessableEntity)
		return
	}
	for _, post := range posts {
		post.Published = post.CreatedAt.Format("Jan 2, 2006")
	}
	if err != nil {
		handleError(w, http.StatusUnprocessableEntity)
		return
	}
	blogData := db.BlogData{
		Posts:   posts,
		Filters: tagFilters,
	}
	html.AllPosts(w, &blogData)
}

func GetPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	post, ok := ctx.Value(postKey).(*db.Post)
	if !ok {
		handleError(w, http.StatusUnprocessableEntity)
		return
	}
	tags, ok := ctx.Value(tagsKey).([]*db.Tag)
	if !ok {
		handleError(w, http.StatusUnprocessableEntity)
		return
	}
	data := db.PostData{
		Post: post,
		Tags: tags,
	}
	html.Post(w, &data)
}

func EditPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	post, ok := ctx.Value(postKey).(*db.Post)
	if !ok {
		handleError(w, http.StatusUnprocessableEntity)
		return
	}
	html.Edit(w, post)
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	// request validation
	err := r.ParseForm()
	if err != nil {
		handleError(w, http.StatusBadRequest)
		return
	}
	username, password := r.FormValue("username"), r.FormValue("password")
	if username == "" || password == "" {
		handleError(w, http.StatusBadRequest)
		return
	}

	var user *db.User

	user, err = db.GetUserByCreds(username, password)
	if !user.IsAdmin {
		handleError(w, http.StatusUnauthorized)
		return
	} else if err != nil {
		handleError(w, http.StatusInternalServerError)
		return
	}
	jwt, err := utils.GenerateToken(user.Id)
	if err != nil {
		handleError(w, http.StatusInternalServerError)
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

// TODO: clean up this logic it is really messy and ugly and i hate it
// TODO: ideally, when you edit the tags, it's reflected in the preview
func HandleUploadMarkdown(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	var buf bytes.Buffer
	file, header, err := r.FormFile("markdown")
	if err != nil {
		handleError(w, http.StatusBadRequest)
		return
	}
	defer file.Close()
	io.Copy(&buf, file)
	contents := buf.String()
	title := utils.FormatTitle(header.Filename)
	slug := utils.TitleToSlug(title)
	tags := utils.ParseTags(contents)
	utils.CleanPostContent(&contents)
	mk, err := markdown.ParseMD(contents)
	if err != nil {
		handleError(w, http.StatusBadRequest)
		return
	}
	html := fmt.Sprintf(`
		<div class="raw-container">
            <h2 id="raw-post-title">Raw</h2>
            <textarea class="raw-post" oninput={previewPostBody(this.value)} name="post-content" form="create-post-form">%s</textarea>
            <form class="upload-markdown-container" enctype="multipart/form-data" hx-post="/markdown" hx-target=".raw-post" hx-swap="innerHTML">
                <input type="file" name="markdown">
                <input type="submit" value="Upload Markdown"></button>
            </form>
            <label for="post-title">Title</label>
            <input type="text" name="post-title" value="%s" oninput={previewPostTitle(this.value)} form="create-post-form">
            <label for="post-slug">Slug</label>
            <input type="text" name="post-slug" value="%s" form="create-post-form">
			<label for="tags">Tags</label>
            <input type="text" name="tags" value="%s" form="create-post-form">
            <form class="create-post-container" id="create-post-form" method="post" action="/post">
                <button type="submit">Create Post</button>
            </form>
        </div>
		<div class="preview-container">
            <h2 id="preview-post-title">Preview</h2>
            <h3 class="preview-title">%s</h3>
            <div class="preview-post">%s</div>
        </div>
	`, mk, title, slug, tags, title, mk)
	w.Write([]byte(html))
}

func HandleCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		handleError(w, http.StatusBadRequest)
		return
	}
	cookie, err := r.Cookie("jwt")
	if err != nil {
		handleError(w, http.StatusInternalServerError)
		return
	}
	tokenString := cookie.Value
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.SignKey), nil
	})
	if err != nil || !token.Valid {
		handleError(w, http.StatusBadRequest)
		return
	}
	content := r.FormValue("post-content")
	tags := strings.Split(r.FormValue("tags"), " ")
	claims := token.Claims.(jwt.MapClaims)
	post := db.Post{
		UserId:    int(claims["user_id"].(float64)), // user_id is a float64 in the map and not an int for some reason
		Title:     r.FormValue("post-title"),
		Slug:      r.FormValue("post-slug"),
		Content:   template.HTML(content),
		Published: time.Now().Format("Monday, January 2, 2006"),
	}
	postID, err := db.CreatePost(&post)
	if err != nil {
		handleError(w, http.StatusInternalServerError)
		return
	}
	err = db.CreateTags(postID, tags)
	if err != nil {
		handleError(w, http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", "/admin")
	w.WriteHeader(http.StatusOK)
}

func HandleDeletePost(w http.ResponseWriter, r *http.Request) {
	if postID := chi.URLParam(r, "postID"); postID != "" {
		postIdInt, err := strconv.Atoi(postID)
		if err != nil {
			handleError(w, http.StatusBadRequest)
			return
		}
		err = db.DeletePost(postIdInt)
		if err != nil {
			handleError(w, http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func HandleEditPost(w http.ResponseWriter, r *http.Request) {
	if postID := chi.URLParam(r, "postID"); postID != "" {
		postIdInt, err := strconv.Atoi(postID)
		if err != nil {
			handleError(w, http.StatusBadRequest)
			return
		}
		err = r.ParseForm()
		if err != nil {
			handleError(w, http.StatusBadRequest)
			return
		}
		post := db.Post{
			Title:     r.FormValue("post-title"),
			Slug:      r.FormValue("post-slug"),
			Content:   template.HTML(r.FormValue("post-content")),
			UpdatedAt: time.Now(),
		}
		err = db.EditPost(postIdInt, &post)
		if err != nil {
			handleError(w, http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("HX-Redirect", "/admin")
	http.Redirect(w, r, "/admin", http.StatusOK)
}

func handleError(w http.ResponseWriter, statusCode int) {
	var statusErr types.StatusError
	switch statusCode {
	case http.StatusNotFound:
		statusErr = types.NewStatusError(errors.New(http.StatusText(http.StatusNotFound)), http.StatusNotFound)
	case http.StatusBadRequest:
		statusErr = types.NewStatusError(errors.New(http.StatusText(http.StatusBadRequest)), http.StatusBadRequest)
	case http.StatusUnauthorized:
		statusErr = types.NewStatusError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	case http.StatusInternalServerError:
		statusErr = types.NewStatusError(errors.New(http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError)
	case http.StatusUnprocessableEntity:
		statusErr = types.NewStatusError(errors.New(http.StatusText(http.StatusUnprocessableEntity)), http.StatusUnprocessableEntity)
	}
	html.Error(w, statusErr)
}
