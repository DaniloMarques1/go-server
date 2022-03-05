package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var (
	portFlag int
	fileFlag string
)

type ErrorDto struct {
	Message string `json:"message"`
}

func main() {
	parseFlags()

	handler, err := NewHandler(fileFlag, portFlag)
	if err != nil {
		log.Fatal(err)
	}

	handler.router.Use(middleware)
	handler.router.NotFound(handleNotFound)

	fmt.Println("Resources available")
	fmt.Println("-----------------------------------------------------------------------")
	for entity := range handler.db {
		resources(entity, handler.serverPort)
		handler.registerRoutes(entity)
	}

	fmt.Println()
	log.Printf("Starting server on port %v\n", handler.serverPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", handler.serverPort), handler.router))
}

// set the serverPort and fileName
func parseFlags() {
	flag.IntVar(&portFlag, "p", 8080, "Defined server port")
	flag.IntVar(&portFlag, "port", 8080, "Defined server port")
	flag.StringVar(&fileFlag, "watch", "db.json", "Defines which file will represent the api database")

	flag.Parse()
}

// it will print the resources available
func resources(entity string, port int) {
	baseUrl := fmt.Sprintf("http://localhost:%v", port)
	fmt.Printf("%v/%v\n", baseUrl, entity)
}

// writes a json to the response writter object
func RespondJSON(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorDto{Message: message})
}

// intercept request and add content type header to it
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// midleware to handle undefined route/endpoint
func handleNotFound(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusNotFound, "endpoint not found")
	return
}
