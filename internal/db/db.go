package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"personal-site/internal/config"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func init() {
	err := connect()
	if err != nil {
		log.Fatal(err)
	}
	pingErr := DB.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")
}

func connect() error {
	var err error
	// if db exists, just connect, otherwise initialize
	if _, err := os.Stat("./db.sqlite"); errors.Is(err, os.ErrNotExist) {
		err = initialize(DB)
		if err != nil {
			return err
		}
	}
	DB, err = sql.Open("sqlite3", "./db.sqlite")
	if err != nil {
		return err
	}
	return nil
}

// TODO: think about security (don't store passwords as plaintext)
func initialize(*sql.DB) error {
	user, pass := config.AdminUser, config.AdminPass
	stmt := `
	CREATE TABLE IF NOT EXISTS user(
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, 
		username VARCHAR(255), 
		password VARCHAR(255), 
		is_admin BOOLEAN
	);
	CREATE TABLE IF NOT EXISTS post(
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, 
		user_id INTEGER,
		title TEXT,
		slug TEXT,
		content TEXT,
		created_at TIMESTAMP,
		updated_at TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES user(id)
	);
	`
	// create user and post tables
	_, err := DB.Exec(stmt)
	if err != nil {
		return err
	}

	insertStmt := `
		INSERT INTO user (username, password, is_admin) VALUES (?, ?, ?);
	`

	// create admin user
	_, err = DB.Exec(insertStmt, user, pass, true)
	if err != nil {
		return err
	}
	return nil
}

func GetPost(post_id int) (*Post, error) {
	var post Post
	row := DB.QueryRow("SELECT * FROM post WHERE id = ?", post_id)
	err := row.Scan(&post.Id, &post.UserId, &post.Title, &post.Slug, &post.Contents, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func GetPostBySlug(slug string) (*Post, error) {
	var post Post
	row := DB.QueryRow("SELECT * FROM post WHERE slug = ?", slug)
	err := row.Scan(&post.Id, &post.UserId, &post.Title, &post.Slug, &post.Contents, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func GetUserByCreds(username string, password string) (*User, error) {
	var user User
	row := DB.QueryRow("SELECT * FROM user WHERE username = ? AND password = ?", username, password)
	err := row.Scan(&user.Id, &user.Username, &user.Password, &user.IsAdmin)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func CreatePost(post *Post) error {
	_, err := DB.Exec(
		"INSERT INTO post (user_id, title, slug, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);",
		post.UserId, post.Title, post.Slug, post.Contents, time.Now(), time.Now())
	if err != nil {
		return err
	}
	return nil
}
