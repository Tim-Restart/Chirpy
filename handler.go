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
	"context"
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

	ctx := r.Context()

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

	chirpyRed, err := cfg.DBQueries.IsChirpyRed(ctx, dbUser.ID)
	if err != nil {
		log.Printf("Error getting user Red stauts: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}


	
	// Do not include a password hash or field here!
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		IsChirpyRed: chirpyRed.Valid && chirpyRed.Bool,
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

	// See if there is an author ID in the query
	// if so, call the function to get only all the chirps form that author
	s := r.URL.Query().Get("author_id")

	if s != "" {
		// Encode the authorID into a UUID
		author, err := uuid.Parse(s)
		if err != nil {
			errResp := errorResponse{
			Error: "Invalid Chirp ID format",
		}
		http.Error(w, "Failed to encode UUID for chirp", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %s", errResp)
		return
		}


		// get the chirps from the author and decode into dbCHirp
		dbChirp, err := cfg.DBQueries.ChirpsFrom(ctx, author)
		if err != nil {

			if err.Error() == "sql: no rows in result set" {
				// Chirp not found, return 404
				w.WriteHeader(http.StatusNotFound)
				return
			}
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
		// Struct to hold the body of the authors chirps
		var selectedChirps []Chirp
		for _, body := range dbChirp {
   			selectedChirps = append(selectedChirps, Chirp{
        	Body: body,
      
    	})
}

		// Respond with above
		err = respondWithJSON(w, http.StatusOK, selectedChirps)
		if err != nil {
		// Handle JSON encoding error
			http.Error(w, "Failed to encode chirps", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %s", err)
			return
	}
		return
	}

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

		if err.Error() == "sql: no rows in result set" {
			// Chirp not found, return 404
			w.WriteHeader(http.StatusNotFound)
			return
		}
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

	ctx := r.Context()

	// Define a struct to take the JSON input
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		//ExpiresInSeconds *int    `json:"expires_in_seconds"`
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
			respondWithError(w, http.StatusInternalServerError, "Failed to encode JSON")
			log.Printf("JSON encoding error: %s", err)
			return
		}
	
			
	}

	expiration := time.Hour
	

	// Start by looking up a user in the DB by their email and return the hash?
	dbUser, err := cfg.DBQueries.GetEmail(ctx, params.Email)
	if err != nil {
		errResp := errorResponse{
			Error: "Incorrect email or password",
		}
		errJ := respondWithJSON(w, 401, errResp)
		if errJ != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to encode JSON")
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
			respondWithError(w, http.StatusInternalServerError, "Failed to encode JSON")
			log.Printf("JSON encoding error: %s", err)
			return
		}
	}

	chirpyRed, err := cfg.DBQueries.IsChirpyRed(ctx, dbUser.ID)
	if err != nil {
		log.Printf("Error getting user Red stauts: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}


	// Assign the user information to be returned on successful password
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		IsChirpyRed: chirpyRed.Valid && chirpyRed.Bool,
		}

	// After validating user credentials
	tokenString, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expiration) // Use your expiration value here
	if err != nil {
		// Handle the error, perhaps return a 500
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	user.Token = tokenString

	user.Refresh_Token, err = auth.MakeRefreshToken() // return the refresh_token here
	err = auth.SaveRefreshToken(user.Refresh_Token, dbUser.ID, *cfg.DBQueries)
	if err != nil {
		log.Print("Error saving refresh token")
		return
	}
	
	// Encode the response and return the results
	err = respondWithJSON(w, 200, user)
	if err != nil {
		// Handle JSON encoding error
		respondWithError(w, http.StatusInternalServerError, "Failed to encode JSON")
		log.Printf("JSON encoding error: %s", err)
		return
	}
	
}

// requires a refresh token to be present in the headers
func (cfg *ApiConfig) refresh(w http.ResponseWriter, r *http.Request) {
	token, _ := auth.GetBearerToken(r.Header)
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "No token provided")
		return
	}

	// Get user info from the refresh token
	rows, err := cfg.DBQueries.GetUserFromRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}
	
	// Check if we got any results
	if len(rows) == 0 {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}
	
	// Use the first row (assuming tokens are unique)
	tokenInfo := rows[0]
	
	// Checks if the token has been revoked
	if tokenInfo.RevokedAt.Valid {
		respondWithError(w, http.StatusUnauthorized, "Token revoked")
		return
	} 

	// Check if the token has expired
	if time.Now().After(tokenInfo.ExpiresAt) {
		respondWithError(w, http.StatusUnauthorized, "Token expired")
		return
	}

	// Generate a new access token for the user
	accessToken, err := auth.MakeJWT(tokenInfo.UserID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create token")
		return
	}
	
	// Respond with the new access token
	respondWithJSON(w, http.StatusOK, map[string]string{
		"token": accessToken,
	})
	}

func (cfg *ApiConfig) revoke(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	// Get the token
	token, _ := auth.GetBearerToken(r.Header)
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "No token provided")
		return
	}

	err := cfg.DBQueries.RevokeToken(ctx, token)
	if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Couldn't revoke token")
        return
    }

	// Set the status code to 204 No Content
	w.WriteHeader(http.StatusNoContent)
	// No need to write any body for 204
}

// Updates user email and password
func (cfg *ApiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	// Assign context
	ctx := context.Background()

	// Define struct to take decoded body
	type UpdateRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		}


	// Get token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No token found")
		return
	}

	// Need to get original email here, then compare it to the Request Email, if differnet
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Unable to get user details")
			return
		}
	

	

	// Create an empty of above
	var params UpdateRequest

	// Decode the inputed JSON response to the memory
	if err = json.NewDecoder(r.Body).Decode(&params); err != nil {
		// Deal with any JSON decoding errors
		respondWithError(w, http.StatusInternalServerError, "Unable to decode details")
		return
	}
		
	// take email and put it in variable to pass in
	newEmail := params.Email
	newPassword, err := auth.HashPassword(params.Password) 
	if err != nil {
		// Prints the error to the terminal
		log.Println("Error hashing password")
		respondWithError(w, http.StatusInternalServerError, "Unable to update password")
		return
	}



	updateParams :=  database.UpdateUserParams {
		ID:        userID,
		Email:     newEmail,
		HashedPassword: newPassword,
	}

	// pass in details to cfg.DBqueries.UpdateUser ($1 user,$2 email,$3 hashedPW)
	err = cfg.DBQueries.UpdateUser(ctx, updateParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update user details")
		return
	}

	//Struct for the JSON response ommitting the password hash
	userReturn := User{
		ID:        userID,
		Email:     newEmail,
	}

	err = respondWithJSON(w, 200, userReturn)
	if err != nil {
		// Handle JSON encoding error
		respondWithError(w, http.StatusInternalServerError, "Failed to encode JSON")
		log.Printf("JSON encoding error: %s", err)
		return
	}

}
	// Deletes a chirp if the user is correct and Chirp ID provided
func (cfg *ApiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	// Chirp ID comes from response body .id  - this retrieves it from the request and parses to UUID
	idString := r.PathValue("chirpID")
	chirpID, err:= uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to decode ChirpID")
		return
	}

	
	// Get the token first using JWT
	// Get token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No token found")
		return
	}

	// Need to get original email here, then compare it to the Request Email, if differnet
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Unable to get user details")
			return
		}
	
	// Get UserID of chirp creator
	chirpUser, err := cfg.DBQueries.GetUserOfChirp(ctx, chirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to decode ChirpID user")
		return
	}

	// Verify token UserID v Chirp UserID
	if chirpUser != userID {
		respondWithError(w, http.StatusForbidden, "Action not authorised")
		return
	}
	// send delete request to db

	err = cfg.DBQueries.DeleteChirp(ctx, chirpID) 
	if err != nil {
		respondWithError(w, http.StatusForbidden, "Action not authorised")
		return
	}

	//err = respondWithJSON(w, 204, "")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("Chirp Deleted\n"))

}


// Accepts a notification from POLKA that payment has been made for Chirpy Red
func (cfg *ApiConfig) chirpyRedUpgrade(w http.ResponseWriter, r *http.Request) {
	
	// Check the API key here
	checkKey, err := GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if checkKey != cfg.polka {
		respondWithError(w, http.StatusUnauthorized, "Incorrect Key")
		return
	}

	ctx := r.Context()

	// Struct to take the Request body
	type UserData struct {
		UserID		string `json:"user_id"`
	}

	type webhookReturn struct {
		Event		string `json:"event"`
		Data		UserData `json:"data"`
	}

	// Decode the user data

	var params webhookReturn

	// Decode the inputed JSON response to the memory
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		// Deal with any JSON decoding errors
		respondWithError(w, http.StatusUnauthorized, "Unable to decode details")
		return
	}

	if params.Event != "user.upgraded" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "UserID not parsed correctly")
		return
	}

	// Check if user exists
	_, err = cfg.DBQueries.CheckUser(ctx, userID) 
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNoContent)
		return

	}
	

	err = cfg.DBQueries.UpgradeToRed(ctx, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upgrade to red")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNoContent)
	return

}



