package main

import (
	"net/http"
)

func (apiCFG *apiConfig) handlerHealth(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
}
