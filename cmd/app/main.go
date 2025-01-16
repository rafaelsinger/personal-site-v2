package main

import (
	"personal-site/internal/db"
	"personal-site/internal/server"
)

// TODO: set up delve for better debugging

func main() {
	defer db.DB.Close()
	server.Start()
}
