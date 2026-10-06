package main

import (
	"log"
	"net/http"
	"os"

	"node-go-labs/lab4/go-service/internal/controller"
	"node-go-labs/lab4/go-service/internal/repository"
	"node-go-labs/lab4/go-service/internal/router"
	"node-go-labs/lab4/go-service/internal/service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	licenseRepository := repository.NewInMemoryLicenseRepository()
	licenseService := service.NewLicenseService(licenseRepository)
	licenseController := controller.NewLicenseController(licenseService)
	handler := router.New(licenseController)

	log.Printf("lab4 license-service (go) listening on %s", port)
	if err := http.ListenAndServe("0.0.0.0:"+port, handler); err != nil {
		log.Fatal(err)
	}
}
