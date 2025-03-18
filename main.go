package main

import "net/http"


func main() {
	// Make a new server
	mux := http.NewServeMux()

	// Create a new Server struct
	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	// Start the server
	server.ListenAndServe()

}