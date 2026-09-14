package main


import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &Server{store: NewStore()}

	log.Printf("license-service (go) listening on %s", port)
	if err := http.ListenAndServe(":"+port, server.routes()); err != nil {
		log.Fatal(err)
	}
}
