package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Parse command line flags
	var port int
	flag.IntVar(&port, "port", 8080, "Port to listen on")
	flag.Parse()

	// Print a welcome message
	fmt.Println("Custom Metrics Example")
	fmt.Println("======================")
	fmt.Printf("Starting server on port %d\n", port)
	fmt.Printf("Visit http://localhost:%d/ to generate metrics\n", port)
	fmt.Printf("Visit http://localhost:%d/metrics to view metrics\n", port)

	// Run the custom metrics example
	CustomMetricsExample()

	// Start the HTTP server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
