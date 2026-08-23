//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Person struct {
	Name  string
	Age   int
	Email string
}

// helperMongoTimeout bounds each MongoDB helper connection, ping, and shutdown call.
const helperMongoTimeout = 10 * time.Second

func main() {
	// Set up client options and connect to MongoDB
	databaseUrl := os.Getenv("DATABASE_URL")
	clientOptions := options.Client().ApplyURI(databaseUrl).
		SetConnectTimeout(helperMongoTimeout).
		SetServerSelectionTimeout(helperMongoTimeout)
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), helperMongoTimeout)
		defer cancel()
		if err := client.Disconnect(disconnectCtx); err != nil {
			log.Fatal(err)
		}
	}()

	// Check the connection
	pingCtx, cancel := context.WithTimeout(context.Background(), helperMongoTimeout)
	defer cancel()
	err = client.Ping(pingCtx, nil)
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
