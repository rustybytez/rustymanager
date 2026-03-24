package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if os.Getenv("AUTH_TOKEN") == "" {
		log.Fatal("AUTH_TOKEN is required")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "rustymanager.db"
	}

	e, err := newApp(dsn)
	if err != nil {
		log.Fatalf("setup: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
