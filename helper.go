package main

import "encoding/json"
import "log"
import "strings"


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
