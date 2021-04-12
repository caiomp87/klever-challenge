package main

import (
	"api/app/pb"
	"api/config"
	"api/controllers"
	"api/db"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	apiPort  string
	cryptoDb *mongo.Collection
	mongoCtx context.Context
)

func init() {
	err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Could not load environment variables: %s", err.Error())
	}

	apiPort = os.Getenv("API_PORT")

	cryptoDb, mongoCtx, err = db.Connect()
	if err != nil {
		log.Fatalf("Could not connect to mongo database: %s", err.Error())
	}
}

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", apiPort))
	if err != nil {
		log.Fatalf("Could not connect on port :%s: %s", apiPort, err.Error())
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	cryptoService := controllers.CryptoServiceServer{
		Db:  cryptoDb,
		Ctx: mongoCtx,
	}
	pb.RegisterCryptoServiceServer(grpcServer, &cryptoService)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	fmt.Printf("Server succesfully started on port :%s\n", apiPort)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	fmt.Println("\nStopping the server...")
	grpcServer.Stop()
	listener.Close()
	fmt.Println("Closing MongoDB connection")
	cryptoDb.Database().Client().Disconnect(mongoCtx)
	fmt.Println("Done.")
}
