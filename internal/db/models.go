package db

import (
	"html/template"
	"time"
)

type User struct {
	Id       int
	Username string
	Password string
	IsAdmin  bool
}

type Post struct {
	Id        int
	UserId    int
	Title     string
	Slug      string
	Contents  template.HTML
	CreatedAt time.Time
	UpdatedAt time.Time
}
