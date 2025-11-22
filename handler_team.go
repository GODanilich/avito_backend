package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type TeamStruct struct {
	TeamName string            `json:"team_name"`
	Members  []UserWithoutTeam `json:"members"`
}

// handlerMakeTransaction handles a POST api/send endpoint.
// Takes json with fields "from", "to", "amount" as a request body
// and returns json representation of created transaction in response
func (apiCFG *apiConfig) handlerAddTeam(w http.ResponseWriter, r *http.Request) {

	// parameters of JSON request body
	type parameters struct {
		Team  string            `json:"team_name"`
		Users []UserWithoutTeam `json:"members"`
	}

	// decoding JSON request body into params
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing JSON", fmt.Sprint(err))
		return
	}

	_, err = apiCFG.DB.GetTeam(r.Context(), params.Team)
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

	err = apiCFG.DB.WithTx(tx).CreateTeam(r.Context(), params.Team)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	for _, user := range params.Users {
		err = apiCFG.DB.WithTx(tx).UpsertUser(r.Context(), database.UpsertUserParams{
			UserID:   user.UserID,
			Username: user.Username,
			TeamName: sql.NullString{
				String: params.Team,
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

	response := struct {
		Team TeamStruct `json:"team"`
	}{Team: TeamStruct{
		TeamName: params.Team,
		Members:  params.Users,
	},
	}

	log.Printf("response is %v", response)

	respondWithJSON(w, http.StatusCreated, response)

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
