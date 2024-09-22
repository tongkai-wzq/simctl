package route

import (
	"simctl/controller"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Reg() *chi.Mux {
	route := chi.NewRouter()
	route.Use(middleware.Logger)
	route.Get("/meals", controller.Meals)
	route.Get("/buy", controller.NewBuy)
	return route
}
