package main

import "net/http"
import "sync/atomic"
import "fmt"
import "encoding/json"
import "log"
import "strings"

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	// Increments the fileserverHits
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment the counter using atomic32 add
		cfg.fileserverHits.Add(1)
		// pass next to ServerHTTP
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
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

func (cfg *apiConfig) metricsResetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Write([]byte("Counter Reset\n"))
}

func badWordReplacement(chirpy string) ([]byte) {
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

	// Assign the cleaned chirp to the struct for JSON marshalling
	cleanedChirp := cleanedBody {
		CleanedBody: modifiedChirp,
	}

	// Marshal the response into JSON and check it works
	jsonResp, err := json.Marshal(cleanedChirp)
			if err != nil {
				log.Printf("Error marshalling JSON %s", err)
				return nil
			}
	return jsonResp
}


func main() {

	// Create an instance of apiConfig
	cfg := apiConfig{}

	// Make a new server
	mux := http.NewServeMux()

	// Assignes fileHandler so that it can be called in mux.Handle
	fileHandler := http.FileServer(http.Dir("./"))

	// Register paths and their handlers
	// FileServer is in http package, Dir converts the '.' to a directory part
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", fileHandler)))

	// vaidates the 140 characters of the chirp
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		
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
				w.WriteHeader(500)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
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
				w.WriteHeader(400)
				w.Write(jsonResp)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			w.Write(jsonResp)
			return
		}

		type cleanedBody struct {
			Cleaned string `json:"cleaned"`
		}

		chirp := badWordReplacement(params.Body)
		
		

		//successChirp := successResponse{
		//	Valid: true,
		//}

		//jsonResp, err := json.Marshal(successChirp)
		//if err != nil {
		//	log.Printf("Error marshalling JSON %s", err)
		//	w.Header().Set("Content-Type", "application/json")
		//	w.WriteHeader(500)
		//	return
		//}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(chirp)
		

	})

	// mux.HandleFunc()

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
		Addr: ":8080",
		Handler: mux,
	}


	// Start the server
	server.ListenAndServe()
	

}