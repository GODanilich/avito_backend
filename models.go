package main

import "GODanilich/avito_backend/internal/database"

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

func dbUserToUser(dbUser database.User) User {
	return User{
		UserID:   dbUser.UserID,
		Username: dbUser.Username,
		TeamName: dbUser.TeamName.String,
		IsActive: dbUser.IsActive,
	}
}

type UserWithoutTeam struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

func dbUserWithoutTeamToUser(dbUser database.User) UserWithoutTeam {
	return UserWithoutTeam{
		UserID:   dbUser.UserID,
		Username: dbUser.Username,
		IsActive: dbUser.IsActive,
	}
}

func dbUsersWithoutTeamToUsers(dbUsers []database.User) (users []UserWithoutTeam) {
	for _, dbUser := range dbUsers {
		users = append(users, dbUserWithoutTeamToUser(dbUser))
	}
	return users
}
