package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

// handlerSetIsActive handles HTTP requests to set a user's active status
// It expects a JSON request body with user_id and is_active fields
func (apiCFG *apiConfig) handlerSetIsActive(w http.ResponseWriter, r *http.Request) {

	// parameters defines the structure of the expected JSON request body
	type parameters struct {
		UserId   string `json:"user_id"`   // Unique identifier for the user
		IsActive bool   `json:"is_active"` // Boolean flag to set user active/inactive status
	}

	// Decode the JSON request body into parameters struct
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// Return 400 Bad Request if JSON parsing fails
		respondWithError(w, http.StatusBadRequest, "Error parsing JSON", fmt.Sprint(err))
		return
	}

	// Validate that user_id is provided
	if params.UserId == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	// Check if user exists before attempting to update
	_, err = apiCFG.DB.GetUserById(r.Context(), params.UserId)
	if err == sql.ErrNoRows {
		// Return 404 Not Found if user doesn't exist
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		// Return 500 Internal Server Error for other database errors
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "internal error")
		return
	}

	// Update user's active status in the database
	user, err := apiCFG.DB.SetUserActive(r.Context(), database.SetUserActiveParams{
		UserID:   params.UserId,
		IsActive: params.IsActive,
	})
	if err != nil {
		// Handle update errors
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "failed to update user")
		}
		return
	}

	// Return 200 OK with the updated user information
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"user": dbUserToUser(user), // Convert database user to API response format
	})
}

// handlerGetReview handles HTTP requests to get pull requests for a reviewer
// It expects a user_id query parameter
func (apiCFG *apiConfig) handlerGetReview(w http.ResponseWriter, r *http.Request) {

	// Extract user_id from query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// Return 400 Bad Request if user_id is missing
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	// Verify that the user exists
	_, err := apiCFG.DB.GetUserById(r.Context(), userID)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "internal error")
		return
	}

	// Retrieve pull requests assigned to the reviewer
	prs, err := apiCFG.DB.GetPRsForReviewer(r.Context(), userID)
	if err != nil && err != sql.ErrNoRows {
		// Return 500 only for actual errors, not for empty results
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "internal error")
		return
	}

	// Ensure prs is never nil to avoid null in JSON response
	if prs == nil {
		prs = []database.GetPRsForReviewerRow{}
	}

	// Structure the response data
	response := struct {
		UserID       string  `json:"user_id"`       // Reviewer's user ID
		PullRequests []PRRow `json:"pull_requests"` // List of pull requests for review
	}{
		UserID:       userID,
		PullRequests: dbPRRowsToPRRows(prs), // Convert database rows to API response format
	}

	// Return 200 OK with the list of pull requests
	respondWithJSON(w, http.StatusOK, response)
}
