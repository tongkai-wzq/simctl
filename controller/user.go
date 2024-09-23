package controller

import (
	"context"
	"net/http"
	"simctl/wechat"

	"github.com/go-chi/render"
)

type userLogin struct {
	Code string `json:"code"`
}

func UserLogin(w http.ResponseWriter, r *http.Request) {
	var form userLogin
	render.DecodeJSON(r.Response.Body, &form)
	if resp, err := wechat.MiniClient.Auth.Session(context.Background(), form.Code); err == nil {
		render.JSON(w, r, resp)
	} else {
		render.JSON(w, r, resp)
	}
}
