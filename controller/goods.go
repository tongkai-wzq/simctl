package controller

import (
	"net/http"
)

func Meals(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}
