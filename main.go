package main

import "net/http"
import "encoding/json"
import "log"
import _ "github.com/lib/pq"
import "os"
import "database/sql"
import "github.com/joho/godotenv"
import "github.com/Tim-Restart/chirpy/internal/database"
import "fmt"
import "sync/atomic"
import "time"
import (
	"github.com/google/uuid"
)

type ApiConfig struct {
	fileserverHits atomic.Int32
	DBQueries      *database.Queries
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

// Struct for incoming JSON posts
type parameters struct {
	Body string `json:"body"`
}

// Successs repsonse struct
type successResponse struct {
	Valid bool `json:"valid"`
}

// Error response struct
type errorResponse struct {
	Error string `json:"error"`
}

func main() {

	// The first section of code below sets the URL by loading the .env file, then connects all SQL stuff, then stores it in a struct.

	// Load the .env file, panic if it doesn't work
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env files")
	}

	platform := os.Getenv("PLATFORM")

	// set the dbURL to the path for the sql database from the .env file
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		panic("DB_URL is not set in the enviroment, check .env file")
	}

	// Open the SQL database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		panic(err)
	}

	// Create a new instance of *database.Queries
	dbQueries := database.New(db)

	// Store it in the apiConfig struct so we have access anywhere
	// Create an instance of apiConfig
	cfg := ApiConfig{
		DBQueries: dbQueries,
		platform:  platform,
	}

	// Make a new server
	mux := http.NewServeMux()

	// Assignes fileHandler so that it can be called in mux.Handle
	fileHandler := http.FileServer(http.Dir("./"))

	// Register paths and their handlers
	// FileServer is in http package, Dir converts the '.' to a directory part
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", fileHandler)))

	// vaidates the 140 characters of the chirp
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			errResp := errorResponse{
				Error: "Something went wrong",
			}

			jsonResp, err := json.Marshal(errResp)
			if err != nil {
				log.Printf("Error marshalling JSON %s", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError) // Status 500
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError) // Status 500
			w.Write(jsonResp)
			return

		}

		// Checks the length of the chirp
		if len(params.Body) > 140 {
			errResp := errorResponse{
				Error: "Chirp is too long",
			}

			jsonResp, err := json.Marshal(errResp)
			if err != nil {
				log.Printf("Error marshalling JSON %s", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest) // Status 400
				w.Write(jsonResp)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest) // Status 400
			w.Write(jsonResp)
			return
		}

		type cleanedBody struct {
			Cleaned string `json:"cleaned"`
		}

		chirp := badWordReplacement(params.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(chirp)

	})

	// mux.HandleFunc()

	// Adds a new user to the database
	mux.HandleFunc("POST /api/users", cfg.addUser)

	// returns the server metrics
	mux.HandleFunc("GET /admin/metrics", cfg.metricsHandler)

	// Resets the server metrics
	mux.HandleFunc("POST /admin/reset", cfg.metricsResetHandler)

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Set the content type header
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		// Write the status code
		w.WriteHeader(http.StatusOK)

		// Write the response body
		w.Write([]byte("OK\n"))

	})

	// Create a new Server struct
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("######## Ready to serve my lord ########")
	// Start the server
	server.ListenAndServe()

}
