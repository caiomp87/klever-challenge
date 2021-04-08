package main

import (
	"api/app/pb"
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

type CryptoServiceServer struct{}

var cryptoDb *mongo.Collection
var mongoCtx context.Context

func main() {
	fmt.Println("Start server on port :50051...")

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Could not connect on port :50051: %s", err.Error())
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	if err != nil {
		log.Fatalf("Could not connect: %s", err.Error())
	}

	fmt.Println("Server listening on port :50051")

	cryptoDb, mongoCtx, err = db.Connect()
	if err != nil {
		log.Fatalf("Could not connect to mongo database: %s", err.Error())
	}

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
	fmt.Println("Server succesfully started on port :50051")

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
