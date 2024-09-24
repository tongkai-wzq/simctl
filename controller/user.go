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
	if resp, err := wechat.MiniClient.Auth.Session(context.Background(), form.Code); err == nil && resp.ErrCode == 0 {
		var user model.User
		if has, err := db.Engine.Where("openid = ?", resp.OpenID).Get(&user); err == nil && !has {
			user.Openid = resp.OpenID
			user.SessionKey = resp.SessionKey
			db.Engine.Insert(&user)
		}
		_, token, _ := TokenAuth.Encode(map[string]any{"userType": "user", "userId": user.Id})
		render.JSON(w, r, map[string]string{"token": token})
	} else {
		render.JSON(w, r, map[string]string{"code": "4001"})
	}
}
