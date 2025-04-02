package main

import "net/http"
import "fmt"
import "encoding/json"
import "log"
import "github.com/Tim-Restart/chirpy/internal/database"
import "github.com/google/uuid"

func (cfg *ApiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	// Increments the fileserverHits
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment the counter using atomic32 add
		cfg.fileserverHits.Add(1)
		// pass next to ServerHTTP
		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	counter := cfg.fileserverHits.Load()

	//HTML template for admin page
	htmlTemplate := `<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
	</html>`

	// Format the above template with the counter in the %d spot
	htmlContent := fmt.Sprintf(htmlTemplate, counter)

	// Sets the header type to HTML
	w.Header().Set("Content-Type", "text/html")

	// Write the html to the response
	fmt.Fprint(w, htmlContent)
}

func (cfg *ApiConfig) metricsResetHandler(w http.ResponseWriter, r *http.Request) {

	// Check if user is dev prior to allowing reset
	if cfg.platform != "dev" {
		// Error for not having the right permission
		errResp := errorResponse{
			Error: "This endpoint only avaliable in development mode",
		}

		jsonResp, err := json.Marshal(errResp)
		if err != nil {
			log.Printf("Error marshalling JSON %s", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError) // 500 error code
			w.Write([]byte(`{"error":"Internal server error"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden) // 403 error code
		w.Write(jsonResp)
		return
	}

	// Calls the SQLC genereated function to delete all users
	err := cfg.DBQueries.DeleteAllUsers(r.Context())
	if err != nil {
		log.Printf("Error deleting users: %s", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to reset database"}`))
		return
	}

	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK) // Status  200
	w.Write([]byte("Counter Reset\n"))
}

// Function to add a new user to the database by email - uses SQL query from SQLC

func (cfg *ApiConfig) addUser(w http.ResponseWriter, r *http.Request) {

	type createUserRequest struct {
		Email string `json:"email"`
	}

	var params createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		errResp := errorResponse{
			Error: "Invalid request body: " + err.Error(),
		}

		jsonResp, err := json.Marshal(errResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(http.StatusInternalServerError) // Status 500
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest) // 400 for client errors
		w.Write(jsonResp)
		return
	}

	dbUser, err := cfg.DBQueries.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Println("Error mapping to database")

		errResp := errorResponse{
			Error: "Error creating user: " + err.Error(),
		}

		jsonResp, err := json.Marshal(errResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(http.StatusInternalServerError) // Status 500
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError) // Status 500
		w.Write(jsonResp)
		return
	}

	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshalling user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Successful response - returns the userJSON marshalled
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 for resource creation
	w.Write(userJSON)

}

func (cfg *ApiConfig) newChirp(w http.ResponseWriter, r *http.Request) {

	type Chirp_Input struct {
		Body    string `json:"body"`
		User_id string `json:"user_id"`
	}

	// Created an empty Chirp struct
	var params Chirp_Input

	// Decode the JSON input and assign it to the Chirp_input struct
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		errResp := errorResponse{
			Error: "Invalid request body: " + err.Error(),
		}

		jsonResp, err := json.Marshal(errResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(http.StatusInternalServerError) // Status 500
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest) // 400 for client errors
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

	// Clean the chirp body
	cleanedBodyBytes := badWordReplacement(params.Body)
	cleanedBody := string(cleanedBodyBytes)
	fmt.Printf(cleanedBody)

	// 4. Parse the User_id string into a UUID
	userUUID, err := uuid.Parse(params.User_id)
	if err != nil {
		errResp := errorResponse{
			Error: "Invalid user ID format",
		}

		jsonResp, err := json.Marshal(errResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResp)
		return
	}

	// Create the NewChirpParams struct
	chirpParams := database.NewChirpParams{
		Body:   cleanedBody,
		UserID: userUUID,
	}

	// Run the newChirp query? and deal with any errors
	// Sends through the JSON input to the query as args

	dbChirp, err := cfg.DBQueries.NewChirp(r.Context(), chirpParams)
	if err != nil {
		log.Println("Error mapping to chirp database: %v", err)

		errResp := errorResponse{
			Error: "Error creating new chirp: " + err.Error(),
		}

		jsonResp, err := json.Marshal(errResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(http.StatusInternalServerError) // Status 500
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError) // Status 500
		w.Write(jsonResp)
		return
	}

	new_Chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		User_ID:   userUUID,
	}

	chirpJSON, err := json.Marshal(new_Chirp)
	if err != nil {
		log.Printf("Error marshalling user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Successful response - returns the userJSON marshalled
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 for resource creation
	w.Write(chirpJSON)

}

// This is the handler function that calls on the DB query to get all the chirps
// This function only creates the JSON response and encodes it, it then returns
// The HTTP responses and JSON

func (cfg *ApiConfig) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	// Use the request's context for the query
	ctx := r.Context()

	// Call the GetChirps function to fetch chirps from the database
	chirps, err := cfg.getChirps(ctx)
	if err != nil {
		// If an error occurred, respond with 500 Internal Server Error
		http.Error(w, "Failed to fetch chirps", http.StatusInternalServerError)
		log.Printf("Database error: %s", err)
		return
	}

	// Set JSON content type for the response
	w.Header().Set("Content-Type", "application/json")

	// Encode the chirps as JSON and write it to the response
	err = json.NewEncoder(w).Encode(chirps)
	if err != nil {
		// Handle JSON encoding failure
		http.Error(w, "Failed to encode chirps", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %s", err)
		return
	}

	// Response is automatically written at this point (status 200 OK by default)
}
