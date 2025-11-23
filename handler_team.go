package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

type TeamStruct struct {
	TeamName string            `json:"team_name"`
	Members  []UserWithoutTeam `json:"members"`
}

func (apiCFG *apiConfig) handlerAddTeam(w http.ResponseWriter, r *http.Request) {

	// parameters of JSON request body
	type requestBody struct {
		TeamName string            `json:"team_name"`
		Members  []UserWithoutTeam `json:"members"`
	}

	// decoding JSON request body into params
	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing JSON", fmt.Sprint(err))
		return
	}

	if params.TeamName == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

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

	_, err = apiCFG.DB.GetTeam(r.Context(), params.TeamName)
	if err == nil {
		respondWithError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
		return
	} else if err != sql.ErrNoRows {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// making db transactions as atomic operation
	tx, err := apiCFG.dbConn.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", "cannot begin tx")
		return
	}

	defer tx.Rollback()

	err = apiCFG.DB.WithTx(tx).CreateTeam(r.Context(), params.TeamName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	for _, user := range params.Members {
		err = apiCFG.DB.WithTx(tx).UpsertUser(r.Context(), database.UpsertUserParams{
			UserID:   user.UserID,
			Username: user.Username,
			TeamName: sql.NullString{
				String: params.TeamName,
				Valid:  true,
			},
			IsActive: user.IsActive,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}

	// Commiting transaction
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"team": params,
	})

}

func (apiCFG *apiConfig) handlerGetTeam(w http.ResponseWriter, r *http.Request) {

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		respondWithError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	team, err := apiCFG.DB.GetTeam(r.Context(), teamName)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "NOT_FOUND", "team not found")
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	users, err := apiCFG.DB.GetTeamMembers(r.Context(), sql.NullString{String: team, Valid: true})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	response := TeamStruct{
		TeamName: team,
		Members:  dbUsersWithoutTeamToUsers(users),
	}

	respondWithJSON(w, http.StatusOK, response)

}
