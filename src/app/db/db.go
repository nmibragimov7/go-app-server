package db

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"os"
	"time"
)

func ConnectDB() *mongo.Client {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	MongoDB := os.Getenv("MONGODB_URL")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(MongoDB))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("MongoDB connect successful: ")
	return client
}

var Client = ConnectDB()

func OpenDatabase(client *mongo.Client, name string) *mongo.Database {
	database := client.Database(name)
	fmt.Println("Database connect successful: ")
	return database
}

func OpenCollection(database *mongo.Database, name string) *mongo.Collection {
	var collection = database.Collection(name)
	return collection
}
