package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func (apiCFG *apiConfig) handlerSetIsActive(w http.ResponseWriter, r *http.Request) {

	// parameters of JSON request body
	type parameters struct {
		UserId   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	// decoding JSON request body into params
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing JSON", fmt.Sprint(err))
		return
	}

	if params.UserId == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	user, err := apiCFG.DB.SetUserActive(r.Context(), database.SetUserActiveParams{
		UserID:   params.UserId,
		IsActive: params.IsActive,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "failed to update user")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"user": dbUserToUser(user),
	})

}
