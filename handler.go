package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"log"
	"github.com/Tim-Restart/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/Tim-Restart/chirpy/internal/auth"
	"time"
)


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
		Email    string `json:"email"`
		Password string `json:"password"`
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

	// Logic to hash the password and return it to upload to the database 

	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		// Prints the error to the terminal
		log.Println("Error hashing password")
		// Creates the error to respond with
		errResp := errorResponse{
			Error: "Error creating user password: " + err.Error(),
		}

		err = respondWithJSON(w, 500, errResp)
		if err != nil {
			// Handle JSON encoding error
			http.Error(w, "Failed to encode password", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %s", err)
			return
		}
		return
	}

	// Create a CreateUserParams struct
	createParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hash,
}

	dbUser, err := cfg.DBQueries.CreateUser(r.Context(), createParams)
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

	
	// Do not include a password hash or field here!
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

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
		return
	}

	userUUID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
		return
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

	// Create the NewChirpParams struct
	chirpParams := database.NewChirpParams{
		Body:   cleanedBody,
		UserID: userUUID,
	}

	// Run the newChirp query? and deal with any errors
	// Sends through the JSON input to the query as args

	dbChirp, err := cfg.DBQueries.NewChirp(r.Context(), chirpParams)
	if err != nil {
		log.Printf("Error mapping to chirp database: %v", err)

		errResp := errorResponse{
			Error: "Error creating new chirp: " + err.Error(),
		}

		err = respondWithJSON(w, 500, errResp)
		if err != nil {
			// Handle JSON encoding error
			http.Error(w, "Failed to encode chirps", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %s", err)
			return
		}
	}

	new_Chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		User_ID:   userUUID,
	}

	// Testing respondWithJSON

	err = respondWithJSON(w, 201, new_Chirp)
	if err != nil {
		// Handle JSON encoding error
		http.Error(w, "Failed to encode chirps", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %s", err)
		return
	}
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

	// Trying out the respondWithJSON helper

	err = respondWithJSON(w, http.StatusOK, chirps)
	if err != nil {
		// Handle JSON encoding error
		http.Error(w, "Failed to encode chirps", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %s", err)
		return
	}

}

// Returns a single chirp by using the UUID of the chirp
// http.Request.PathValue used in here to do something with a string

func (cfg *ApiConfig) getChirp(w http.ResponseWriter, r *http.Request) {

	// Ok something here about using the UUID to do a query on the DB?
	// Don't forget context...
	idString := r.PathValue("chirpID")
	chirpToGet, err:= uuid.Parse(idString)
	if err != nil {
		errResp := errorResponse{
			Error: "Invalid Chirp ID format",
		}
		http.Error(w, "Failed to encode UUID for chirp", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %s", errResp)
		return
	}
	// maybe assign variable here for the chirp ID?


	dbChirp, err := cfg.DBQueries.GetChirp(r.Context(), chirpToGet)
	if err != nil {
		log.Printf("Error finding chirp in database: %v", err)

		errResp := errorResponse{
			Error: "Error finding chirp: " + err.Error(),
		}

		err = respondWithJSON(w, 500, errResp)
		if err != nil {
			// Handle JSON encoding error
			http.Error(w, "Failed to encode chirps", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %s", err)
			return
		}
	}

	new_Chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		User_ID:   dbChirp.UserID,
	}

	err = respondWithJSON(w, http.StatusOK, new_Chirp)
	if err != nil {
		// Handle JSON encoding error
		http.Error(w, "Failed to encode chirps", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %s", err)
		return
	}

}

func (cfg *ApiConfig) login(w http.ResponseWriter, r *http.Request) {

	// Define a struct to take the JSON input
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		ExpiresInSeconds *int    `json:"expires_in_seconds"`
	}

	// Create an empty of above
	var params LoginRequest
	// Decode the inputed JSON response to the memory
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		// Deal with any JSON decoding errors
		errResp := errorResponse{
			Error: "Invalid request body: " + err.Error(),
		}
		// Respond using the helper function
		errJ := respondWithJSON(w, 500, errResp)
		if errJ != nil {
			// Handle JSON encoding error
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %s", err)
			return
		}
	
			
	}

	var expiration time.Duration
		if params.ExpiresInSeconds == nil {
			expiration = time.Hour // Default to 1 hour if field is missing
		} else if *params.ExpiresInSeconds > 3600 {
			expiration = time.Hour // Cap at 1 hour if client requests more
		} else {
			expiration = time.Duration(*params.ExpiresInSeconds) * time.Second
		}

	// Start by looking up a user in the DB by their email and return the hash?
	dbUser, err := cfg.DBQueries.GetEmail(r.Context(), params.Email)
	if err != nil {
		errResp := errorResponse{
			Error: "Incorrect email or password",
		}
		errJ := respondWithJSON(w, 401, errResp)
		if errJ != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %s", errJ)
		}
		return
	}

	// use the hash and the new password to send to the comparer

	err = auth.CheckPasswordHash(dbUser.HashedPassword, params.Password)
	if err != nil {
		errResp := errorResponse{
			Error: "Incorrect email or password",
		}
		errJ := respondWithJSON(w, 401, errResp)
		if errJ != nil {
			// Handle JSON encoding error
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %s", err)
			return
		}
	}


	// Assign the user information to be returned on successful password
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		}

	// After validating user credentials
	tokenString, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expiration) // Use your expiration value here
	if err != nil {
		// Handle the error, perhaps return a 500
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	user.Token = tokenString
	
	// Encode the response and return the results
	err = respondWithJSON(w, 200, user)
	if err != nil {
		// Handle JSON encoding error
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %s", err)
		return
	}
	
}



