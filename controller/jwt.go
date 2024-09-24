package controller

import "github.com/go-chi/jwtauth/v5"

var TokenAuth *jwtauth.JWTAuth

func init() {
	TokenAuth = jwtauth.New("HS256", []byte("9z0273dlc3qi9"), nil)
}
