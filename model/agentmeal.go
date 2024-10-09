package model

import (
	"simctl/db"

	"xorm.io/builder"
)

type AgentGroup struct {
	Id         int64        `json:"id"`
	AgentId    int64        `json:"agentId"`
	Agent      *Agent       `xorm:"-" json:"agent"`
	GroupId    int64        `json:"groupId"`
	Group      *Group       `xorm:"-" json:"group"`
	Rebates    bool         `json:"rebates"`
	AgentMeals []*AgentMeal `xorm:"-" json:"agentMeals"`
}

func (ag *AgentGroup) LoadGroup() {
	ag.Group = new(Group)
	db.Engine.ID(ag.GroupId).Get(ag.Group)
}

func (ag *AgentGroup) LoadAgentMeals() {
	mealIds := make([]int64, 0, 30)
	db.Engine.Table("meal").Where("group_id", ag.GroupId).Cols("id").Find(&mealIds)
	cond := builder.Eq{"agent_id": ag.AgentId}.And(builder.In("meal_id", mealIds))
	ag.AgentMeals = make([]*AgentMeal, 0, 30)
	db.Engine.Where(cond).Find(ag.AgentMeals)
}

func (ag *AgentGroup) AttachPrice(saleMeals []*SaleMeal) []*SaleMeal {
	var newSaleMeals []*SaleMeal
	for _, saleMeal := range saleMeals {
		for _, agentMeal := range ag.AgentMeals {
			if agentMeal.MealId != saleMeal.MealId {
				continue
			}
			if agentMeal.Price > 0 {
				saleMeal.Price = agentMeal.Price
			}
			newSaleMeals = append(newSaleMeals, saleMeal)
			break
		}
	}
	return newSaleMeals
}

type AgentMeal struct {
	Id       int64   `json:"id"`
	AgentId  int64   `json:"agentId"`
	Agent    *Agent  `xorm:"-" json:"agent"`
	MealId   int64   `json:"mealId"`
	Meal     *Meal   `xorm:"-" json:"meal"`
	StlPrice float64 `json:"stlPrice"`
	Price    float64 `json:"price"`
}

func (am *AgentMeal) LoadAgent() {
	am.Agent = new(Agent)
	db.Engine.ID(am.AgentId).Get(am.Agent)
}

func (am *AgentMeal) LoadMeal() {
	am.Meal = new(Meal)
	db.Engine.ID(am.MealId).Get(am.Meal)
}
