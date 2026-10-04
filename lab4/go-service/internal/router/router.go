package router

import (
	"net/http"

	"node-go-labs/lab4/go-service/internal/controller"
	"node-go-labs/lab4/go-service/internal/httpjson"
	"node-go-labs/lab4/go-service/internal/middleware"
)

func New(controller *controller.UserController) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", controller.Health)
	mux.HandleFunc("GET /api/users", controller.List)
	mux.HandleFunc("GET /api/users/{id}", controller.Get)
	mux.HandleFunc("POST /api/users", controller.Create)
	mux.HandleFunc("PUT /api/users/{id}", controller.Update)
	mux.HandleFunc("DELETE /api/users/{id}", controller.Delete)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpjson.WriteError(w, http.StatusNotFound, "route_not_found", "Route was not found")
	})
	return middleware.Logging(middleware.ValidateJSON(mux))
}
