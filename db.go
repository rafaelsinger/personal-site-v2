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
			content TEXT,
			FOREIGN KEY(user_id) REFERENCES user(id)
	)
	`

	_, err = db.Exec(stmt)
	if err != nil {
		return nil, err
	}

	return db, nil
}
