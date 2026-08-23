//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// helperMongoTimeout bounds each MongoDB helper connection, ping, drop, and shutdown call.
const helperMongoTimeout = 10 * time.Second

func main() {
	// Set up MongoDB client
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

	pingCtx, cancel := context.WithTimeout(context.Background(), helperMongoTimeout)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		log.Fatal(err)
	}

	// Access database
	db := client.Database("testdb")

	// Drop the database
	dropCtx, cancel := context.WithTimeout(context.Background(), helperMongoTimeout)
	defer cancel()
	err = db.Drop(dropCtx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("dropped")
}
