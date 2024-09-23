package route

import (
	"simctl/controller"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func Reg() *chi.Mux {
	route := chi.NewRouter()
	route.Use(middleware.Logger)
	route.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
	}))
	route.Get("/meals", controller.Meals)
	route.Get("/buy", controller.NewBuy)
	route.Post("/userLogin", controller.UserLogin)
	return route
}
