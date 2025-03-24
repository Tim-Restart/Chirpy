package main

import "net/http"
import "sync/atomic"
import "fmt"
import "github.com/Tim-Restart/chirpy/internal/database"


type ApiConfig struct {
	fileserverHits atomic.Int32
	DBQueries *database.Queries
}

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
	cfg.fileserverHits.Store(0)
	w.Write([]byte("Counter Reset\n"))
}