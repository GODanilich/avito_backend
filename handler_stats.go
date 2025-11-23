package main

import (
	"GODanilich/avito_backend/internal/database"
	"fmt"
	"net/http"
)

type StatsResponse struct {
	PRStats         []PRStatusCount   `json:"pr_stats"`
	AssignmentStats []UserAssignCount `json:"assignment_stats"`
}

type PRStatusCount struct {
	Status database.PrStatus `json:"status"`
	Count  int64             `json:"count"`
}

type UserAssignCount struct {
	UserID string `json:"user_id"`
	Count  int64  `json:"count"`
}

func (api *apiConfig) handlerGetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Получаем статистику PR по статусам
	prStatsRaw, err := api.DB.GetPRStats(ctx)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", fmt.Sprintf("cannot fetch PR stats: %v", err))
		return
	}

	// Получаем статистику назначений
	assignStatsRaw, err := api.DB.GetAssignmentStats(ctx)
	if err != nil {
		respondWithError(w, 500, "DB_ERROR", fmt.Sprintf("cannot fetch assignment stats: %v", err))
		return
	}

	// Преобразуем в DTO
	prStats := make([]PRStatusCount, len(prStatsRaw))
	for i, s := range prStatsRaw {
		prStats[i] = PRStatusCount{
			Status: s.Status,
			Count:  s.Count,
		}
	}

	assignmentStats := make([]UserAssignCount, len(assignStatsRaw))
	for i, s := range assignStatsRaw {
		assignmentStats[i] = UserAssignCount{
			UserID: s.UserID,
			Count:  s.Count,
		}
	}

	resp := StatsResponse{
		PRStats:         prStats,
		AssignmentStats: assignmentStats,
	}

	respondWithJSON(w, 200, resp)
}
