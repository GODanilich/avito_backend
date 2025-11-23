package main

import (
	"GODanilich/avito_backend/internal/database"
	"fmt"
	"net/http"
)

// StatsResponse defines the structure for the statistics API response
// It contains two main sections: PR statistics and assignment statistics
type StatsResponse struct {
	PRStats         []PRStatusCount   `json:"pr_stats"`         // Statistics of pull requests grouped by status
	AssignmentStats []UserAssignCount `json:"assignment_stats"` // Statistics of user assignments count
}

// PRStatusCount represents the count of pull requests for a specific status
type PRStatusCount struct {
	Status database.PrStatus `json:"status"` // PR status (e.g., OPEN, MERGED, CLOSED)
	Count  int64             `json:"count"`  // Number of PRs with this status
}

// UserAssignCount represents the number of PR assignments per user
type UserAssignCount struct {
	UserID string `json:"user_id"` // Unique identifier of the user
	Count  int64  `json:"count"`   // Number of PR assignments for this user
}

// handlerGetStats handles HTTP GET requests to retrieve system statistics
// Returns aggregated data about PR statuses and user assignment counts
func (api *apiConfig) handlerGetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get PR statistics grouped by status from the database
	// This typically returns counts for each PR status (OPEN, MERGED, CLOSED, etc.)
	prStatsRaw, err := api.DB.GetPRStats(ctx)
	if err != nil {
		// Return 500 Internal Server Error if database query fails
		respondWithError(w, 500, "DB_ERROR", fmt.Sprintf("cannot fetch PR stats: %v", err))
		return
	}

	// Get assignment statistics - counts of how many PRs each user is assigned to review
	assignStatsRaw, err := api.DB.GetAssignmentStats(ctx)
	if err != nil {
		// Return 500 Internal Server Error if database query fails
		respondWithError(w, 500, "DB_ERROR", fmt.Sprintf("cannot fetch assignment stats: %v", err))
		return
	}

	// Convert database PR stats to API response format (DTO pattern)
	// This provides a clean separation between database and API models
	prStats := make([]PRStatusCount, len(prStatsRaw))
	for i, s := range prStatsRaw {
		prStats[i] = PRStatusCount{
			Status: s.Status, // PR status from database
			Count:  s.Count,  // Count for this status
		}
	}

	// Convert database assignment stats to API response format
	assignmentStats := make([]UserAssignCount, len(assignStatsRaw))
	for i, s := range assignStatsRaw {
		assignmentStats[i] = UserAssignCount{
			UserID: s.UserID, // User identifier
			Count:  s.Count,  // Number of assignments for this user
		}
	}

	// Build the complete response structure
	resp := StatsResponse{
		PRStats:         prStats,         // PR status distribution
		AssignmentStats: assignmentStats, // User assignment counts
	}

	// Return 200 OK with the statistics data
	respondWithJSON(w, 200, resp)
}
