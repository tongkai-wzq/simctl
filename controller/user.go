package controller

import (
	"context"
	"net/http"
	"simctl/db"
	"simctl/model"
	"simctl/wechat"

	"github.com/go-chi/render"
)

type userLogin struct {
	Code string `json:"code"`
}

func UserLogin(w http.ResponseWriter, r *http.Request) {
	var form userLogin
	render.DecodeJSON(r.Body, &form)
	if resp, err := wechat.MiniClient.Auth.Session(context.Background(), form.Code); err == nil {
		var user model.User
		if has, err := db.Engine.Where("openid = ?", resp.OpenID).Get(&user); err == nil && !has {
			user.Openid = resp.OpenID
			db.Engine.Insert(&user)
		}
		render.JSON(w, r, user)
	} else {
		render.JSON(w, r, nil)
	}
}
