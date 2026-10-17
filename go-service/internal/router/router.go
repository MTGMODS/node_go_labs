package router

import (
	"net/http"

	"node-go-labs/go-service/internal/controller"
	"node-go-labs/go-service/internal/httpjson"
	"node-go-labs/go-service/internal/middleware"
)

type Dependencies struct {
	StreamController  *controller.StreamController
	ExportController  *controller.ExportController
	EventController   *controller.EventController
	AccountController *controller.AccountController
	WebSocketHandler  http.Handler
	RealtimeStats     http.HandlerFunc
}

func New(controller *controller.LicenseController, dependencies ...Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", controller.Health)
	mux.HandleFunc("GET /licenses", controller.List)
	mux.HandleFunc("GET /licenses/{id}", controller.Get)
	mux.HandleFunc("POST /licenses", controller.Create)
	mux.HandleFunc("PUT /licenses/{id}", controller.Update)
	mux.HandleFunc("DELETE /licenses/{id}", controller.Delete)
	mux.HandleFunc("GET /api/licenses", controller.List)
	mux.HandleFunc("GET /api/licenses/{id}", controller.Get)
	mux.HandleFunc("POST /api/licenses", controller.Create)
	mux.HandleFunc("PUT /api/licenses/{id}", controller.Update)
	mux.HandleFunc("DELETE /api/licenses/{id}", controller.Delete)
	if len(dependencies) > 0 {
		dependency := dependencies[0]
		if dependency.ExportController != nil {
			mux.HandleFunc("GET /api/licenses/export", dependency.ExportController.Export)
		}
		if dependency.EventController != nil {
			mux.HandleFunc("GET /api/events", dependency.EventController.List)
		}
		if dependency.AccountController != nil {
			mux.HandleFunc("GET /api/accounts", dependency.AccountController.List)
			mux.HandleFunc("POST /api/accounts/transfer", dependency.AccountController.Transfer)
		}
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
