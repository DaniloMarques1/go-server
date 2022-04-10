package cmd

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	portFlag int
	fileFlag string
	minified bool
	help     bool
)

// messages

const (
	portMessage     = "Defined server port"
	fileMessage     = "Defines which file will represent the api database"
	minifiedMessage = "Indicates whether json should be written in one line or not"
	helpMessage     = "Show the usage of the go-server"
)

func Run() {
	parseFlags()
	if help {
		usage()
		os.Exit(1)
	}

	handler, err := NewHandler(fileFlag, portFlag, minified)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Resources available")
	fmt.Println("-----------------------------------------------------------------------")
	for entity := range handler.db {
		resources(entity, handler.serverPort)
		handler.RegisterRoutes(entity)
	}

	fmt.Println()
	log.Printf("Starting server on port %v\n", handler.serverPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", handler.serverPort), handler.router))
}

// set the serverPort and fileName
func parseFlags() {
	flag.IntVar(&portFlag, "port", 8080, portMessage)
	flag.StringVar(&fileFlag, "watch", "db.json", fileMessage)
	flag.BoolVar(&minified, "minified", false, minifiedMessage)
	flag.BoolVar(&help, "help", false, helpMessage)

	flag.Parse()
}

// it will print the resources available
func resources(entity string, port int) {
	baseUrl := fmt.Sprintf("http://localhost:%v", port)
	fmt.Printf("%v/%v\n", baseUrl, entity)
}

// show all the options that can be used with the go-server cli
func usage() {
	fmt.Println("All the options available")
	fmt.Println()
	fmt.Printf("-watch:            %v\n", fileMessage)
	fmt.Printf("-port:             %v\n", portMessage)
	fmt.Printf("-minified:         %v\n", minifiedMessage)
	fmt.Printf("-help:             %v\n", helpMessage)
}
