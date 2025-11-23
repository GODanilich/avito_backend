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

	_, err = apiCFG.DB.GetUserById(r.Context(), params.UserId)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "internal error")
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

func (apiCFG *apiConfig) handlerGetReview(w http.ResponseWriter, r *http.Request) {

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	_, err := apiCFG.DB.GetUserById(r.Context(), userID)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "internal error")
		return
	}

	prs, err := apiCFG.DB.GetPRsForReviewer(r.Context(), userID)
	if err != nil && err != sql.ErrNoRows {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "internal error")
		return
	}
	if prs == nil {
		prs = []database.GetPRsForReviewerRow{}
	}

	response := struct {
		UserID       string  `json:"user_id"`
		PullRequests []PRRow `json:"pull_requests"`
	}{
		UserID:       userID,
		PullRequests: dbPRRowsToPRRows(prs),
	}

	respondWithJSON(w, http.StatusOK, response)

}
