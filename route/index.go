package route

import (
	"simctl/controller"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth/v5"
)

func Reg() *chi.Mux {
	route := chi.NewRouter()
	route.Use(middleware.Logger)
	route.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
	}))
	route.Get("/meals", controller.Meals)
	route.Post("/userLogin", controller.UserLogin)
	route.Group(func(route chi.Router) {
		route.Use(jwtauth.Verifier(controller.TokenAuth))
		route.Use(jwtauth.Authenticator(controller.TokenAuth))
		route.Get("/buy", controller.NewBuy)
	})
	return route
}
