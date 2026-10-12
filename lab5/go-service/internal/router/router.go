package router

import (
	"net/http"

	"node-go-labs/lab5/go-service/internal/controller"
	"node-go-labs/lab5/go-service/internal/httpjson"
	"node-go-labs/lab5/go-service/internal/middleware"
)

type Dependencies struct {
	StreamController *controller.StreamController
	WebSocketHandler http.Handler
	RealtimeStats    http.HandlerFunc
}

func New(controller *controller.LicenseController, dependencies ...Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", controller.Health)
	mux.HandleFunc("GET /licenses", controller.List)
	mux.HandleFunc("GET /licenses/{id}", controller.Get)
	mux.HandleFunc("POST /licenses", controller.Create)
	mux.HandleFunc("PUT /licenses/{id}", controller.Update)
	mux.HandleFunc("DELETE /licenses/{id}", controller.Delete)
	if len(dependencies) > 0 {
		dependency := dependencies[0]
		if dependency.StreamController != nil {
			mux.HandleFunc("GET /streams/source", dependency.StreamController.SourceInfo)
			mux.HandleFunc("POST /streams/source", dependency.StreamController.GenerateSource)
			mux.HandleFunc("POST /streams/process", dependency.StreamController.Start)
			mux.HandleFunc("GET /streams/jobs", dependency.StreamController.ListJobs)
			mux.HandleFunc("GET /streams/jobs/{id}", dependency.StreamController.GetJob)
		}
		if dependency.WebSocketHandler != nil {
			mux.Handle("GET /ws", dependency.WebSocketHandler)
		}
		if dependency.RealtimeStats != nil {
			mux.HandleFunc("GET /realtime/clients", dependency.RealtimeStats)
		}
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.WriteError(w, http.StatusNotFound, "route_not_found", "Route was not found")
	})
	return middleware.Logging(middleware.ValidateJSON(mux))
}
