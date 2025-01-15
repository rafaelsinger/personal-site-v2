package main

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func Connect() (*sql.DB, error) {
	var err error
	os.Remove("./db.sqlite")
	db, err = sql.Open("sqlite3", "./db.sqlite")
	if err != nil {
		return nil, err
	}
	err = initialize(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// TODO: think about security (don't store passwords as plaintext)
func initialize(*sql.DB) error {
	user, pass := os.Getenv("ADMIN_USER"), os.Getenv("ADMIN_PASS")
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
	_, err := db.Exec(stmt)
	if err != nil {
		return err
	}

	insertStmt := `
		INSERT INTO user (username, password, is_admin) VALUES (?, ?, ?);
	`

	// create admin user
	_, err = db.Exec(insertStmt, user, pass, true)
	if err != nil {
		return err
	}
	return nil
}
