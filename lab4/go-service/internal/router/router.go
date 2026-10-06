package router

import (
	"net/http"

	"node-go-labs/lab4/go-service/internal/controller"
	"node-go-labs/lab4/go-service/internal/httpjson"
	"node-go-labs/lab4/go-service/internal/middleware"
)

func New(controller *controller.LicenseController) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", controller.Health)
	mux.HandleFunc("GET /licenses", controller.List)
	mux.HandleFunc("GET /licenses/{id}", controller.Get)
	mux.HandleFunc("POST /licenses", controller.Create)
	mux.HandleFunc("PUT /licenses/{id}", controller.Update)
	mux.HandleFunc("DELETE /licenses/{id}", controller.Delete)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.WriteError(w, http.StatusNotFound, "route_not_found", "Route was not found")
	})
	return middleware.Logging(middleware.ValidateJSON(mux))
}
