package main

import (
	"log"
	"net/http"
	"os"

	"node-go-labs/lab5/go-service/internal/controller"
	"node-go-labs/lab5/go-service/internal/realtime"
	"node-go-labs/lab5/go-service/internal/repository"
	"node-go-labs/lab5/go-service/internal/router"
	"node-go-labs/lab5/go-service/internal/service"
	"node-go-labs/lab5/go-service/internal/streaming"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	licenseRepository := repository.NewInMemoryLicenseRepository()
	licenseService := service.NewLicenseService(licenseRepository)
	licenseController := controller.NewLicenseController(licenseService)
	hub := realtime.NewHub(5)
	go hub.Run()
	realtimeHandler := realtime.NewHandler(hub)
	streamService := streaming.NewService(os.Getenv("DATA_DIR"), hub.BroadcastJSON)
	streamController := controller.NewStreamController(streamService)
	handler := router.New(licenseController, router.Dependencies{
		StreamController: streamController,
		WebSocketHandler: http.HandlerFunc(realtimeHandler.ServeWS),
		RealtimeStats:    realtimeHandler.Stats,
	})

	log.Printf("lab5 license-stream-service (go) listening on %s", port)
	if err := http.ListenAndServe("0.0.0.0:"+port, handler); err != nil {
		log.Fatal(err)
	}
}
