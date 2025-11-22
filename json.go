package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, status_code int, code, msg string) {
	if status_code > 499 {
		log.Printf("Responding with %v error: %v", status_code, msg)
	}
	type errResponse struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	response := struct {
		Error errResponse `json:"error"`
	}{
		Error: errResponse{
			Message: msg,
			Code:    code,
		},
	}
	respondWithJSON(w, status_code, response)
}

// json responder
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(payload); err != nil {
		log.Printf("Failed to encode JSON response: %v, payload: %v", err, payload)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}
}
