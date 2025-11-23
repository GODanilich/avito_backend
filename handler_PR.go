package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"time"
)

type mergePrResponseStruct struct {
	PullRequestID     string            `json:"pull_request_id"`
	PullRequestName   string            `json:"pull_request_name"`
	AuthorID          string            `json:"author_id"`
	Status            database.PrStatus `json:"status"`
	AssignedReviewers []string          `json:"assigned_reviewers"`
	MergedAt          string            `json:"mergedAt"`
}
type rPrResponseStruct struct {
	PullRequestID     string            `json:"pull_request_id"`
	PullRequestName   string            `json:"pull_request_name"`
	AuthorID          string            `json:"author_id"`
	Status            database.PrStatus `json:"status"`
	AssignedReviewers []string          `json:"assigned_reviewers"`
}

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

	if params.PullRequestID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id is required")
		return
	}
	if params.PullRequestName == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_name is required")
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

	teamName := author.TeamName

	// searching for active users
	candidates, err := api.DB.GetActiveReviewersForTeam(ctx, database.GetActiveReviewersForTeamParams{
		TeamName: teamName,
		UserID:   params.AuthorID,
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
		Status:            "OPEN",
		AssignedReviewers: reviewers,
	}

	respondWithJSON(w, 201, map[string]interface{}{
		"pr": response,
	})
}

func (api *apiConfig) handlerMergePR(w http.ResponseWriter, r *http.Request) {
	var params struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	if params.PullRequestID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id is required")
		return
	}
	ctx := r.Context()

	// check if PR exists
	pr, err := api.DB.GetPR(ctx, params.PullRequestID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "NOT_FOUND", "PR not found")
		} else {
			respondWithError(w, 500, "DB_ERROR", err.Error())
		}
		return
	}

	if pr.Status != "MERGED" {
		pr, err = api.DB.SetPRMerged(ctx, params.PullRequestID)
		if err != nil {
			respondWithError(w, 500, "DB_ERROR", err.Error())
			return
		}
	}

	reviewers, err := api.DB.GetPRReviewers(ctx, params.PullRequestID)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"pr": mergePrResponseStruct{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            pr.Status,
			AssignedReviewers: reviewers,
			MergedAt:          pr.MergedAt.Time.Format(time.RFC3339),
		},
	})
}

func (api *apiConfig) handlerReassignPR(w http.ResponseWriter, r *http.Request) {
	var params struct {
		PullRequestID string `json:"pull_request_id"`
		OldreviewerID string `json:"old_reviewer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, 400, "BAD_REQUEST", "invalid json")
		return
	}

	if params.PullRequestID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id is required")
		return
	}
	if params.OldreviewerID == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "old_reviewer_id is required")
		return
	}

	ctx := r.Context()

	// 1. Проверяем, что PR существует
	pr, err := api.DB.GetPR(ctx, params.PullRequestID)
	if err == sql.ErrNoRows {
		respondWithError(w, 404, "NOT_FOUND", "PR not found")
		return
	}
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Проверка статуса PR
	if pr.Status == "MERGED" {
		respondWithError(w, 409, "PR_MERGED", "cannot reassign on merged PR")
		return
	}

	// 2. Проверяем, что старый ревьювер действительно назначен на этот PR
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

	// 2. Загружаем автора
	author, err := api.DB.GetUserById(ctx, pr.AuthorID)
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

	team := author.TeamName

	// 3. Кандидаты на reassignment
	candidates, err := api.DB.GetEligibleReassignReviewers(ctx, database.GetEligibleReassignReviewersParams{
		TeamName:      team,
		UserID:        params.OldreviewerID,
		PullRequestID: params.PullRequestID,
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}
	// Проверяем, что есть кандидаты
	if len(candidates) == 0 {
		respondWithError(w, 409, "NO_CANDIDATE", "no active replacement candidate in team")
		return
	}
	// выберем до двух
	newReviewer := chooseRandomReviewers(candidates, 1)[0]

	// 4. Транзакция: удаляем старых и создаём новых
	tx, err := api.dbConn.BeginTx(ctx, nil)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", "cannot begin tx")
		return
	}
	defer tx.Rollback()

	qtx := api.DB.WithTx(tx)

	// удалить старых
	if err := qtx.DeleteReviewer(ctx, database.DeleteReviewerParams{
		PullRequestID: params.PullRequestID,
		UserID:        params.OldreviewerID,
	}); err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// добавить новых
	err = qtx.AddReviewer(ctx, database.AddReviewerParams{
		PullRequestID: params.PullRequestID,
		UserID:        newReviewer,
	})
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	// 6. Получаем обновленный список ревьюверов для ответа
	updatedReviewers, err := api.DB.GetPRReviewers(ctx, params.PullRequestID)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	pr, err = api.DB.GetPR(ctx, params.PullRequestID)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", err.Error())
		return
	}

	response := struct {
		PR         rPrResponseStruct `json:"pr"`
		ReplacedBy string            `json:"replaced_by"`
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
