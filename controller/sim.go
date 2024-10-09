package controller

import (
	"net/http"
	"simctl/db"
	"simctl/model"

	"github.com/go-chi/render"
	"xorm.io/builder"
)

type simDtl struct {
	Id       int64  `json:"id"`
	Iccid    string `json:"iccid"`
	Msisdn   string `json:"msisdn"`
	MapNber  string `json:"mapNber"`
	Operator string `json:"operator"`
}

func Sim(w http.ResponseWriter, r *http.Request) {
	var cond builder.Cond
	if nber := r.URL.Query().Get("nber"); nber != "" {
		cond = builder.Eq{"iccid": nber}.Or(builder.Eq{"msisdn": nber}).Or(builder.Eq{"map_nber": nber})
	}
	var sim model.Sim
	if has, err := db.Engine.Where(cond).Get(&sim); err == nil && has {
		render.JSON(w, r, map[string]any{"code": 0, "id": sim.Id})
	} else if err == nil {
		render.JSON(w, r, map[string]any{"code": 4003, "msg": "未查询到"})
	}
}

func SimList(w http.ResponseWriter, r *http.Request) {

}
