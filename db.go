package main

import (
	"database/sql"
)

func connect() (*sql.DB, error) {
	var err error
	db, err = sql.Open("sqlite3", "./data.sqlite")
	if err != nil {
		return nil, err
	}

	// sqlStmt := `
	// create table if not exists articles (id integer not null primary key autoincrement, title text, content text);
	// `

	// _, err = db.Exec(sqlStmt)
	// if err != nil {
	// 	return nil, err
	// }

	return db, nil
}
