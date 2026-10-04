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

	userRepository := repository.NewInMemoryUserRepository()
	userService := service.NewUserService(userRepository)
	userController := controller.NewUserController(userService)
	handler := router.New(userController)

	log.Printf("lab4 go service listening on %s", port)
	if err := http.ListenAndServe("0.0.0.0:"+port, handler); err != nil {
		log.Fatal(err)
	}
}
