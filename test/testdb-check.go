package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Person struct {
	Name  string
	Age   int
	Email string
}

func main() {
	godotenv.Load(".env.test")

	// Set up client options and connect to MongoDB
	databaseUrl := os.Getenv("DATABASE_URL")
	clientOptions := options.Client().ApplyURI(databaseUrl)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	// Access a database and collection
	database := client.Database("testdb")
	collection := database.Collection("testcollection")

	// Create a context with a timeout of 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Define a filter to check if the email already exists
	filter := bson.D{{"email", "john.doe@example.com"}}

	// Attempt to find a document with the given email
	var existingPerson Person
	err = collection.FindOne(ctx, filter).Decode(&existingPerson)

	if err == mongo.ErrNoDocuments {
		fmt.Println("notfound")
	} else if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("found")
	}
}
