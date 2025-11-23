package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

// TeamStruct represents the structure of a team with its members
type TeamStruct struct {
	TeamName string            `json:"team_name"` // Name of the team
	Members  []UserWithoutTeam `json:"members"`   // List of team members
}

// handlerAddTeam handles HTTP POST requests to create a new team
// It creates a team and adds all specified members to it in a transactional manner
func (apiCFG *apiConfig) handlerAddTeam(w http.ResponseWriter, r *http.Request) {

	// requestBody defines the structure of the expected JSON request
	type requestBody struct {
		TeamName string            `json:"team_name"` // Name of the team to create
		Members  []UserWithoutTeam `json:"members"`   // List of users to add to the team
	}

	// Decode the JSON request body into the params struct
	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err := decoder.Decode(&params)
	if err != nil {
		// Return 400 Bad Request if JSON parsing fails
		respondWithError(w, http.StatusBadRequest, "Error parsing JSON", fmt.Sprint(err))
		return
	}

	// Validate that team_name is provided and not empty
	if params.TeamName == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	// Validate each member in the members list
	for _, user := range params.Members {
		if user.UserID == "" {
			respondWithError(w, http.StatusBadRequest, "INVALID_USER_ID", "user_id cannot be empty")
			return
		}
		if user.Username == "" {
			respondWithError(w, http.StatusBadRequest, "INVALID_USERNAME", "username cannot be empty")
			return
		}
	}

	// Check if a team with the same name already exists
	_, err = apiCFG.DB.GetTeam(r.Context(), params.TeamName)
	if err == nil {
		// Team already exists - return error
		respondWithError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
		return
	} else if err != sql.ErrNoRows {
		// Some other database error occurred
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Start a database transaction to ensure atomicity
	// This ensures either all operations succeed or all fail
	tx, err := apiCFG.dbConn.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "cannot begin tx")
		return
	}

	// Ensure transaction is rolled back if not committed
	defer tx.Rollback()

	// Create the team in the database
	err = apiCFG.DB.WithTx(tx).CreateTeam(r.Context(), params.TeamName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Add each member to the team
	for _, user := range params.Members {
		err = apiCFG.DB.WithTx(tx).UpsertUser(r.Context(), database.UpsertUserParams{
			UserID:   user.UserID,
			Username: user.Username,
			TeamName: sql.NullString{
				String: params.TeamName, // Set the team name for this user
				Valid:  true,            // Mark that team name is set
			},
			IsActive: user.IsActive,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}

	// Commit the transaction - all operations succeed
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Return 201 Created with the team details
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"team": params,
	})
}

// handlerGetTeam handles HTTP GET requests to retrieve team information
// It returns the team details along with all its members
func (apiCFG *apiConfig) handlerGetTeam(w http.ResponseWriter, r *http.Request) {

	// Extract team_name from query parameters
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	// Verify that the team exists
	team, err := apiCFG.DB.GetTeam(r.Context(), teamName)
	if err == sql.ErrNoRows {
		// Team not found - return 404
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", "team not found")
		return
	} else if err != nil {
		// Other database error
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Retrieve all members of the team
	users, err := apiCFG.DB.GetTeamMembers(r.Context(), sql.NullString{
		String: team,
		Valid:  true,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Structure the response data
	response := TeamStruct{
		TeamName: team,
		Members:  dbUsersWithoutTeamToUsers(users), // Convert database users to API response format
	}

	// Return 200 OK with team information
	respondWithJSON(w, http.StatusOK, response)
}
