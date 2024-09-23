package main

import (
	"fmt"
	"net/http"
	"simctl/db"
	"simctl/model"
	"simctl/route"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if err := db.Engine.Sync(new(model.Agent), new(model.User), new(model.Meal), new(model.Group), new(model.AgentGroup), new(model.AgentMeal), new(model.Sim), new(model.Order)); err != nil {
		fmt.Println("database sync", err.Error())
	}
	http.ListenAndServe(":3000", route.Reg())
}
