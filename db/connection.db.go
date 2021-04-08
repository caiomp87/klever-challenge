package db

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connect() (*mongo.Collection, context.Context, error) {
	var dbName = os.Getenv("DB_NAME")
	var dbCollection = os.Getenv("DB_COLLECTION")
	var dbHost = os.Getenv("DB_HOST")
	var dbPort = os.Getenv("DB_PORT")

	fmt.Println("Connecting to MongoDB...")
	mongoCtx := context.Background()

	connectionString := fmt.Sprintf("mongodb://%s:%s", dbHost, dbPort)
	db, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(connectionString))
	if err != nil {
		return nil, mongoCtx, err
	}

	err = db.Ping(mongoCtx, nil)
	if err != nil {
		return nil, mongoCtx, err
	}

	fmt.Printf("Connected to MongoDB: %s\n", connectionString)
	cryptoDb := db.Database(dbName).Collection(dbCollection)

	return cryptoDb, mongoCtx, nil
}
