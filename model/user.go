package model

type User struct {
	Id         int64  `json:"id"`
	Name       string `json:"name"`
	Mobile     string `json:"mobile"`
	Openid     string `json:"openid"`
	SessionKey string `json:"sessionKey"`
}
