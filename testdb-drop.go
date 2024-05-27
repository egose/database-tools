package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env.test")

	// Set up MongoDB client
	databaseUrl := os.Getenv("DATABASE_URL")
	clientOptions := options.Client().ApplyURI(databaseUrl)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// Access database
	db := client.Database("testdb")

	// Drop the database
	err = db.Drop(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("dropped")
}
