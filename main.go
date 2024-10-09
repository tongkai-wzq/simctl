package main

import (
	"log"
	"net/http"
	"simctl/db"
	"simctl/model"
	"simctl/route"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if err := db.Engine.Sync(new(model.Agent), new(model.User), new(model.Meal), new(model.Group), new(model.AgentGroup), new(model.AgentMeal), new(model.Sim), new(model.GatewayUser), new(model.Order), new(model.Packet), new(model.Rebates)); err != nil {
		log.Println("database sync", err.Error())
	}
	db.Engine.Iterate(new(model.GatewayUser), func(i int, bean interface{}) error {
		gwUser := bean.(*model.GatewayUser)
		model.GatewayUsers[gwUser.Id] = gwUser
		return nil
	})
	http.ListenAndServe(":3000", route.Reg())
}
