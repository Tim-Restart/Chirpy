package main

import "net/http"


func main() {
	// Make a new server
	mux := http.NewServeMux()

	// Register paths and their handlers
	// FileServer is in http package, Dir converts the '.' to a directory part
	mux.Handle("/", http.FileServer(http.Dir(".")))

	// Create a new Server struct
	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	// Start the server
	server.ListenAndServe()

}