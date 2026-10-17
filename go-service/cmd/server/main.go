package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"node-go-labs/go-service/internal/controller"
	"node-go-labs/go-service/internal/db"
	"node-go-labs/go-service/internal/events"
	"node-go-labs/go-service/internal/realtime"
	"node-go-labs/go-service/internal/repository"
	"node-go-labs/go-service/internal/router"
	"node-go-labs/go-service/internal/service"
	"node-go-labs/go-service/internal/streaming"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	postgres, err := db.OpenPostgres()
	if err != nil {
		log.Fatalf("connect PostgreSQL: %v", err)
	}
	defer postgres.Close()
	mongoClient, mongoDatabase, err := db.OpenMongo()
	if err != nil {
		log.Fatalf("connect MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	licenseRepository, err := repository.NewFromType(os.Getenv("REPO_TYPE"), postgres, mongoDatabase)
	if err != nil {
		log.Fatal(err)
	}
	hub := realtime.NewHub(5)
	go hub.Run()
	eventLogger := events.NewMongoLogger(mongoDatabase)
	publish := func(event service.Event) error {
		if err := eventLogger.Log(event); err != nil {
			return err
		}
		hub.BroadcastJSON(event)
		return nil
	}
	licenseService := service.NewLicenseService(licenseRepository, publish)
	licenseController := controller.NewLicenseController(licenseService)
	realtimeHandler := realtime.NewHandler(hub)
	streamService := streaming.NewService(os.Getenv("DATA_DIR"), hub.BroadcastJSON)
	streamController := controller.NewStreamController(streamService)
	batchReader, ok := licenseRepository.(repository.LicenseBatchReader)
	if !ok {
		log.Fatal("selected repository does not support batch export")
	}
	exportService := streaming.NewDatabaseExportService(batchReader, os.Getenv("DATA_DIR"), hub.BroadcastJSON)
	handler := router.New(licenseController, router.Dependencies{
		StreamController:  streamController,
		ExportController:  controller.NewExportController(exportService),
		EventController:   controller.NewEventController(eventLogger),
		AccountController: controller.NewAccountController(service.NewAccountService(postgres)),
		WebSocketHandler:  http.HandlerFunc(realtimeHandler.ServeWS),
		RealtimeStats:     realtimeHandler.Stats,
	})

	server := &http.Server{Addr: "0.0.0.0:" + port, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("lab6 license persistence service (go, %s) listening on %s", os.Getenv("REPO_TYPE"), port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownContext)
}
