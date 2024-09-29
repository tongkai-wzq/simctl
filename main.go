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
	if err := db.Engine.Sync(new(model.Agent), new(model.User), new(model.Meal), new(model.Group), new(model.AgentGroup), new(model.AgentMeal), new(model.Sim), new(model.Order), new(model.Packet), new(model.Rebates)); err != nil {
		log.Println("database sync", err.Error())
	}
	http.ListenAndServe(":3000", route.Reg())
}
