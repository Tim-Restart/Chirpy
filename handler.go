package main

import "net/http"
import "fmt"
import "encoding/json"
import "log"


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
		ID:			dbUser.ID,
		CreatedAt:	dbUser.CreatedAt,
		UpdatedAt:	dbUser.UpdatedAt,
		Email:		dbUser.Email,
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

