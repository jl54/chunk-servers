package main

import (
	"log"

	"github.com/jl54/chunk-servers/internal/cli"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	cli.Handle()
}
