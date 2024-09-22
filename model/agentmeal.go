package model

type AgentGroup struct {
	Id      int64
	AgentId int64  `json:"agentId"`
	Agent   *Agent `xorm:"-" json:"agent"`
	GroupId int64  `json:"groupId"`
	Group   *Group `xorm:"-" json:"group"`
	Rebates bool   `json:"rebates"`
}

type AgentMeal struct {
	Id       int64
	AgentId  int64   `json:"agentId"`
	Agent    *Agent  `xorm:"-" json:"agent"`
	MealId   int64   `json:"mealId"`
	Meal     *Meal   `xorm:"-" json:"meal"`
	StlPrice float64 `json:"stlPrice"`
	Price    float64 `json:"price"`
}
