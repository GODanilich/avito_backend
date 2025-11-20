package main

import (
	"GODanilich/avito_backend/internal/database"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

// API config
type apiConfig struct {
	DB     *database.Queries
	dbConn *sql.DB
}

func main() {
	// load environment variables from .env
	godotenv.Load(".env")

	// getting PORT from .env
	portString := os.Getenv("PORT")
	if portString == "" {
		log.Fatal("PORT is not found in the environment")
	}

	// getting DB_URL from .env
	db_URL := os.Getenv("DB_URL")
	if db_URL == "" {
		log.Fatal("DB_URL is not found in the environment")
	}

	// connecting to db
	conn, err := sql.Open("postgres", db_URL)
	if err != nil {
		log.Fatal("Can`t connect to database:", err)
	}

	defer conn.Close()

	db := database.New(conn)

	apiCFG := apiConfig{
		DB:     db,
		dbConn: conn,
	}

	// routing conf
	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	v1Router := chi.NewRouter()

	v1Router.Get("/api/health", apiCFG.handlerHealth)

	router.Mount("/v1", v1Router)

	// configuring HTTP server
	srv := &http.Server{
		Handler: router,
		Addr:    ":" + portString,
	}
	log.Printf("Server is starting on port %v", portString)
	// starting the server
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("PORT is:", portString)
}
