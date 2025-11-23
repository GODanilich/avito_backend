package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"time"
)

// mergePrResponseStruct defines the response structure for merged pull requests
type mergePrResponseStruct struct {
	PullRequestID     string            `json:"pull_request_id"`    // Unique identifier for the PR
	PullRequestName   string            `json:"pull_request_name"`  // Name/title of the PR
	AuthorID          string            `json:"author_id"`          // ID of the PR author
	Status            database.PrStatus `json:"status"`             // Current status of the PR
	AssignedReviewers []string          `json:"assigned_reviewers"` // List of reviewer IDs
	MergedAt          string            `json:"mergedAt"`           // Timestamp when PR was merged
}

// rPrResponseStruct defines the response structure for reassigned pull requests
type rPrResponseStruct struct {
	PullRequestID     string            `json:"pull_request_id"`    // Unique identifier for the PR
	PullRequestName   string            `json:"pull_request_name"`  // Name/title of the PR
	AuthorID          string            `json:"author_id"`          // ID of the PR author
	Status            database.PrStatus `json:"status"`             // Current status of the PR
	AssignedReviewers []string          `json:"assigned_reviewers"` // List of reviewer IDs
}

// handlerCreatePR handles HTTP POST requests to create a new pull request
// It creates a PR and automatically assigns random reviewers from the author's team
func (api *apiConfig) handlerCreatePR(w http.ResponseWriter, r *http.Request) {
	// Define the expected request parameters
	var params struct {
		PullRequestID   string `json:"pull_request_id"`   // Unique identifier for the PR
		PullRequestName string `json:"pull_request_name"` // Name/title of the PR
		AuthorID        string `json:"author_id"`         // ID of the user creating the PR
	}

	// Decode JSON request body
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, 400, "BAD_REQUEST", "invalid json")
		return
	}

	// Validate required fields
	if params.PullRequestID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id is required")
		return
	}
	if params.PullRequestName == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_name is required")
		return
	}

	ctx := r.Context()

	// Check if PR with the same ID already exists
	if _, err := api.DB.GetPR(ctx, params.PullRequestID); err == nil {
		respondWithError(w, 409, "PR_EXISTS", "PR id already exists")
		return
	} else if err != sql.ErrNoRows {
		// Database error other than "not found"
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Verify that the author exists
	author, err := api.DB.GetUserById(ctx, params.AuthorID)
	if err == sql.ErrNoRows {
		respondWithError(w, 404, "NOT_FOUND", "author not found")
		return
	}
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Verify that the author belongs to a team
	if !author.TeamName.Valid {
		respondWithError(w, 404, "NOT_FOUND", "author has no team")
		return
	}

	teamName := author.TeamName

	// Find active reviewers in the same team (excluding the author)
	candidates, err := api.DB.GetActiveReviewersForTeam(ctx, database.GetActiveReviewersForTeamParams{
		TeamName: teamName,
		UserID:   params.AuthorID, // Exclude the author from reviewers
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Randomly select 2 reviewers from available candidates
	reviewers := chooseRandomReviewers(candidates, 2)

	// Start database transaction to ensure atomic operations
	tx, err := api.dbConn.BeginTx(ctx, nil)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", "cannot begin tx")
		return
	}
	defer tx.Rollback() // Ensure rollback if transaction fails

	qtx := api.DB.WithTx(tx) // Create query interface with transaction

	// Create the pull request in database
	err = qtx.CreatePR(ctx, database.CreatePRParams{
		PullRequestID:   params.PullRequestID,
		PullRequestName: params.PullRequestName,
		AuthorID:        params.AuthorID,
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Assign selected reviewers to the PR
	for _, rID := range reviewers {
		err = qtx.AddReviewer(ctx, database.AddReviewerParams{
			PullRequestID: params.PullRequestID,
			UserID:        rID,
		})
		if err != nil {
			respondWithError(w, 500, "DB_ERROR", err.Error())
			return
		}
	}

	// Commit the transaction - all operations succeed
	if err := tx.Commit(); err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Prepare success response
	response := struct {
		PullRequestID     string   `json:"pull_request_id"`
		PullRequestName   string   `json:"pull_request_name"`
		AuthorID          string   `json:"author_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	}{
		PullRequestID:     params.PullRequestID,
		PullRequestName:   params.PullRequestName,
		AuthorID:          params.AuthorID,
		Status:            "OPEN", // New PRs are created with OPEN status
		AssignedReviewers: reviewers,
	}

	// Return 201 Created with PR details
	respondWithJSON(w, 201, map[string]interface{}{
		"pr": response,
	})
}

// handlerMergePR handles HTTP POST requests to merge a pull request
// It updates the PR status to MERGED and records the merge timestamp
func (api *apiConfig) handlerMergePR(w http.ResponseWriter, r *http.Request) {
	var params struct {
		PullRequestID string `json:"pull_request_id"` // ID of the PR to merge
	}

	// Decode JSON request body
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	// Validate required field
	if params.PullRequestID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id is required")
		return
	}
	ctx := r.Context()

	// Check if PR exists
	pr, err := api.DB.GetPR(ctx, params.PullRequestID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "NOT_FOUND", "PR not found")
		} else {
			respondWithError(w, 500, "DB_ERROR", err.Error())
		}
		return
	}

	// Update PR status to MERGED if not already merged
	if pr.Status != "MERGED" {
		pr, err = api.DB.SetPRMerged(ctx, params.PullRequestID)
		if err != nil {
			respondWithError(w, 500, "DB_ERROR", err.Error())
			return
		}
	}

	// Get the list of reviewers assigned to this PR
	reviewers, err := api.DB.GetPRReviewers(ctx, params.PullRequestID)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Prepare and send success response
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"pr": mergePrResponseStruct{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            pr.Status,
			AssignedReviewers: reviewers,
			MergedAt:          pr.MergedAt.Time.Format(time.RFC3339), // Format timestamp as RFC3339
		},
	})
}

// handlerReassignPR handles HTTP POST requests to reassign a reviewer on a pull request
// It replaces an existing reviewer with a new random reviewer from the same team
func (api *apiConfig) handlerReassignPR(w http.ResponseWriter, r *http.Request) {
	var params struct {
		PullRequestID string `json:"pull_request_id"` // ID of the PR
		OldreviewerID string `json:"old_reviewer_id"` // ID of the reviewer to replace
	}

	// Decode JSON request body
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, 400, "BAD_REQUEST", "invalid json")
		return
	}

	// Validate required fields
	if params.PullRequestID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id is required")
		return
	}
	if params.OldreviewerID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "old_reviewer_id is required")
		return
	}

	ctx := r.Context()

	// 1. Check if PR exists
	pr, err := api.DB.GetPR(ctx, params.PullRequestID)
	if err == sql.ErrNoRows {
		respondWithError(w, 404, "NOT_FOUND", "PR not found")
		return
	}
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Check PR status - cannot reassign on merged PRs
	if pr.Status == "MERGED" {
		respondWithError(w, 409, "PR_MERGED", "cannot reassign on merged PR")
		return
	}

	// 2. Verify that the old reviewer is actually assigned to this PR
	isAssigned, err := api.DB.IsReviewerAssigned(ctx, database.IsReviewerAssignedParams{
		PullRequestID: params.PullRequestID,
		UserID:        params.OldreviewerID,
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}
	if !isAssigned {
		respondWithError(w, 409, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
		return
	}

	// 3. Load author information to determine team
	author, err := api.DB.GetUserById(ctx, pr.AuthorID)
	if err == sql.ErrNoRows {
		respondWithError(w, 404, "NOT_FOUND", "author not found")
		return
	}
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Verify author has a team
	if !author.TeamName.Valid {
		respondWithError(w, 404, "NOT_FOUND", "author has no team")
		return
	}

	team := author.TeamName

	// 4. Find eligible replacement reviewers
	candidates, err := api.DB.GetEligibleReassignReviewers(ctx, database.GetEligibleReassignReviewersParams{
		TeamName:      team,
		UserID:        params.OldreviewerID, // Exclude the old reviewer
		PullRequestID: params.PullRequestID, // Exclude current reviewers
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Check if there are available candidates
	if len(candidates) == 0 {
		respondWithError(w, 409, "NO_CANDIDATE", "no active replacement candidate in team")
		return
	}

	// Randomly select one new reviewer
	newReviewer := chooseRandomReviewers(candidates, 1)[0]

	// 5. Transaction: remove old reviewer and add new one
	tx, err := api.dbConn.BeginTx(ctx, nil)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", "cannot begin tx")
		return
	}
	defer tx.Rollback()

	qtx := api.DB.WithTx(tx)

	// Remove the old reviewer
	if err := qtx.DeleteReviewer(ctx, database.DeleteReviewerParams{
		PullRequestID: params.PullRequestID,
		UserID:        params.OldreviewerID,
	}); err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Add the new reviewer
	err = qtx.AddReviewer(ctx, database.AddReviewerParams{
		PullRequestID: params.PullRequestID,
		UserID:        newReviewer,
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// 6. Get updated list of reviewers for the response
	updatedReviewers, err := api.DB.GetPRReviewers(ctx, params.PullRequestID)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Get updated PR information
	pr, err = api.DB.GetPR(ctx, params.PullRequestID)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Prepare success response
	response := struct {
		PR         rPrResponseStruct `json:"pr"`          // Updated PR information
		ReplacedBy string            `json:"replaced_by"` // ID of the new reviewer
	}{
		PR: rPrResponseStruct{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            pr.Status,
			AssignedReviewers: updatedReviewers,
		},
		ReplacedBy: newReviewer,
	}

	respondWithJSON(w, 200, response)
}

// chooseRandomReviewers randomly selects reviewers from the candidate list
// count: number of reviewers to select
// Returns: slice of selected reviewer IDs
func chooseRandomReviewers(candidates []string, count int) []string {
	n := len(candidates)
	if n == 0 {
		return []string{} // No candidates available
	}
	if n <= count {
		return candidates // Not enough candidates, return all available
	}

	// Shuffle the candidates array randomly
	rand.Shuffle(n, func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Return the first 'count' elements after shuffling
	return candidates[:count]
}
