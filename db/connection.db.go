package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connect() (*mongo.Collection, context.Context, error) {
	fmt.Println("Connecting to MongoDB...")
	mongoCtx := context.Background()

	db, err := mongo.Connect(mongoCtx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, mongoCtx, err
	}

	err = db.Ping(mongoCtx, nil)
	if err != nil {
		return nil, mongoCtx, err
	}

	fmt.Println("Connected to MongoDB")
	cryptoDb := db.Database("klever").Collection("cryptos")

	return cryptoDb, mongoCtx, nil
}
