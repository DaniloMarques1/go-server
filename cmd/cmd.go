package cmd

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var (
	portFlag int
	fileFlag string
)

func Run() {
	parseFlags()

	handler, err := NewHandler(fileFlag, portFlag)
	if err != nil {
		log.Fatal(err)
	}

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
