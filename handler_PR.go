package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"net/http"
)

func (api *apiConfig) handlerCreatePR(w http.ResponseWriter, r *http.Request) {
	var params struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, 400, "BAD_REQUEST", "invalid json")
		return
	}

	ctx := r.Context()

	// check if PR already exists
	if _, err := api.DB.GetPR(ctx, params.PullRequestID); err == nil {
		respondWithError(w, 409, "PR_EXISTS", "PR id already exists")
		return
	} else if err != sql.ErrNoRows {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// check if author exists
	author, err := api.DB.GetUserById(ctx, params.AuthorID)
	if err == sql.ErrNoRows {
		respondWithError(w, 404, "NOT_FOUND", "author not found")
		return
	}
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	if !author.TeamName.Valid {
		respondWithError(w, 404, "NOT_FOUND", "author has no team")
		return
	}

	teamName := author.TeamName.String

	// searching for active users
	candidates, err := api.DB.GetActiveReviewersForTeam(ctx, database.GetActiveReviewersForTeamParams{
		TeamName: sql.NullString{
			String: teamName,
			Valid:  true,
		},
		UserID: params.AuthorID,
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// mixing reviewers
	reviewers := chooseRandomReviewers(candidates, 2)

	// transaction
	tx, err := api.dbConn.BeginTx(ctx, nil)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", "cannot begin tx")
		return
	}
	defer tx.Rollback()

	qtx := api.DB.WithTx(tx)

	// creating PR
	err = qtx.CreatePR(ctx, database.CreatePRParams{
		PullRequestID:   params.PullRequestID,
		PullRequestName: params.PullRequestName,
		AuthorID:        params.AuthorID,
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// adding reviewers
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

	if err := tx.Commit(); err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	resp := map[string]interface{}{
		"pr": map[string]interface{}{
			"pull_request_id":    params.PullRequestID,
			"pull_request_name":  params.PullRequestName,
			"author_id":          params.AuthorID,
			"status":             "OPEN",
			"assigned_reviewers": reviewers,
		},
	}

	respondWithJSON(w, 201, resp)
}

func (api *apiConfig) handlerMergePR(w http.ResponseWriter, r *http.Request) {
	var params struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	ctx := r.Context()

	// check if PR exists
	_, err := api.DB.GetPR(ctx, params.PullRequestID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "NOT_FOUND", "PR not found")
		} else {
			respondWithError(w, 500, "DB_ERROR", err.Error())
		}
		return
	}

	pr, err := api.DB.SetPRMerged(ctx, params.PullRequestID)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	reviewers, err := api.DB.GetPRReviewers(ctx, params.PullRequestID)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	resp := map[string]interface{}{
		"pr": map[string]interface{}{
			"pull_request_id":    pr.PullRequestID,
			"pull_request_name":  pr.PullRequestName,
			"author_id":          pr.AuthorID,
			"status":             pr.Status,
			"assigned_reviewers": reviewers,
		},
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func chooseRandomReviewers(candidates []string, count int) []string {
	n := len(candidates)
	if n == 0 {
		return []string{}
	}
	if n <= count {
		return candidates
	}

	rand.Shuffle(n, func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	return candidates[:count]
}
