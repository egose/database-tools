package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/joho/godotenv"
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

	// Insert a document only if the email doesn't already exist
	person := Person{
		Name:  "John Doe",
		Age:   30,
		Email: "john.doe@example.com",
	}

	// Define a filter to check if the email already exists
	filter := bson.D{{"email", person.Email}}

	// Attempt to find a document with the given email
	var existingPerson Person
	err = collection.FindOne(ctx, filter).Decode(&existingPerson)

	// If no document is found, insert the new person
	if err == mongo.ErrNoDocuments {
		_, err = collection.InsertOne(ctx, person)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Document inserted successfully!")
		fmt.Println("ready")
	} else if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Document with the same email already exists.")
		fmt.Println("ready")
	}
}
