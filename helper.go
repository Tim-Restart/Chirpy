package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"errors"
)

func badWordReplacement(chirpy string) string {
	//Struct for formating the JSON response
	type cleanedBody struct {
		CleanedBody string `json:"cleaned_body"`
	}

	// Assign params body to a variable so I can keep overwriting it (not sure if thats a good idea)
	// not needed for this way - chirp := params.Body
	chirp := strings.Split(chirpy, " ")
	for i := 0; i < len(chirp); i++ {
		if strings.ToLower(chirp[i]) == "kerfuffle" {
			chirp[i] = "****"
		} else if strings.ToLower(chirp[i]) == "sharbert" {
			chirp[i] = "****"
		} else if strings.ToLower(chirp[i]) == "fornax" {
			chirp[i] = "****"
		} else {
			continue
		}
	}

	// Joins the array back together
	modifiedChirp := strings.Join(chirp, " ")

	return modifiedChirp

	// Assign the cleaned chirp to the struct for JSON marshalling
	//cleanedChirp := cleanedBody{
	//	CleanedBody: modifiedChirp,
	//}

	// Marshal the response into JSON and check it works
	//jsonResp, err := json.Marshal(cleanedChirp)
	//if err != nil {
	//	log.Printf("Error marshalling JSON %s", err)
	//	return nil
	//}
	//return jsonResp
}

// Returns all chirps in order by created_at
// This is then called by the handler function to create the response

func (cfg *ApiConfig) getChirps(ctx context.Context) ([]Chirp, error) {

	// Fetch chirps from the database
	chirpsFromDB, err := cfg.DBQueries.GetChirps(ctx)
	if err != nil {
		log.Print("Failed to fetch chirps")
		return nil, err
	}

	// Transform the results if necessary
	chirpsResponse := []Chirp{}
	for _, chirp := range chirpsFromDB {
		chirpsResponse = append(chirpsResponse, Chirp{
			ID:        chirp.ID,        // Assuming ID is a UUID and needs conversion
			CreatedAt: chirp.CreatedAt, // Timestamp
			UpdatedAt: chirp.UpdatedAt, // Timestamp
			Body:      chirp.Body,      // The body of the chirp
			User_ID:   chirp.UserID,    // Assuming UserID is a UUID
		})

		// Return the slice of chirps and nil error on success

	}
	return chirpsResponse, nil
}

// Helper function to handle all JSON encoding

func respondWithJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return err
	}
	return nil
}

func respondWithError(w http.ResponseWriter, status int, message string) {
    type errorResponse struct {
        Error string `json:"error"`
    }
    
    response := errorResponse{
        Error: message,
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(response)
    // Note: We're not returning an error here since we can't do much with it anyway
    // after we've already started writing the response
}

// Function to get API key from header

func GetAPIKey(headers http.Header) (string, error) {
	
	// Gets the bearer token and does stuff with it
	authHeader := headers.Get("Authorization")
	
	if authHeader == "" {
		log.Print("Error: Authorization header is empty")
		return "", errors.New("authorization header is missing")
	}
	
	if !strings.HasPrefix(authHeader, "ApiKey ") {
		log.Print("Error: Authorization header does not start with 'ApiKey '")
		return "", errors.New("invalid authorization header format")
	}

	polkaKey := strings.TrimPrefix(authHeader, "ApiKey ")

	if strings.TrimSpace(polkaKey) == "" {
		log.Print("Error: Key is empty after trimming")
		return "", errors.New("authorization token is empty")
	}

	return polkaKey, nil
}

