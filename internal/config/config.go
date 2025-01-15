package config

import (
	"fmt"
	"log"
	"os"

	"github.com/go-chi/jwtauth/v5"
	"github.com/joho/godotenv"
)

var IsDev bool
var Addr string
var Port string
var SignKey []byte
var TokenAuth *jwtauth.JWTAuth
var AdminUser string
var AdminPass string

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	IsDev = os.Getenv("GO_ENV") == "development"
	Addr = os.Getenv("SERVER_ADDR")
	Port = fmt.Sprintf(":%s", os.Getenv("SERVER_PORT"))
	SignKey = []byte(os.Getenv("SIGN_KEY"))
	TokenAuth = jwtauth.New("HS256", SignKey, nil)
	AdminUser = os.Getenv("ADMIN_USER")
	AdminPass = os.Getenv("ADMIN_PASS")
}
