package main

import (
	"api/app/pb"
	"api/db"
	"api/models"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"gopkg.in/mgo.v2/bson"
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
	pb.RegisterCryptoServiceServer(grpcServer, &CryptoServiceServer{})
	if err != nil {
		log.Fatalf("Could not connect: %s", err.Error())
	}

	fmt.Println("Server listening on port :50051")

	cryptoDb, mongoCtx, err = db.Connect()
	if err != nil {
		log.Fatalf("Could not connect to mongo database: %s", err.Error())
	}

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

func (s *CryptoServiceServer) CreateCrypto(ctx context.Context, req *pb.CreateCryptoRequest) (*pb.CreateCryptoResponse, error) {
	data := models.CryptoItem{
		Name:        strings.ToUpper(req.GetName()),
		Description: strings.Title(req.GetDescription()),
		Likes:       0,
		Dislikes:    0,
	}

	result, err := cryptoDb.InsertOne(mongoCtx, data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}

	data.Id = result.InsertedID.(primitive.ObjectID)

	return &pb.CreateCryptoResponse{Success: true}, nil
}

func (s *CryptoServiceServer) ListCryptos(req *pb.ListCryptosRequest, stream pb.CryptoService_ListCryptosServer) error {
	data := &models.CryptoItem{}

	cursor, err := cryptoDb.Find(context.Background(), bson.M{})
	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		err := cursor.Decode(data)
		if err != nil {
			return status.Errorf(codes.Unavailable, fmt.Sprintf("Could not decode data: %v", err))
		}

		stream.Send(&pb.ListCryptosResponse{
			Crypto: &pb.Crypto{
				Id:          data.Id.Hex(),
				Name:        data.Name,
				Description: data.Description,
				Likes:       strconv.Itoa(data.Likes),
				Dislikes:    strconv.Itoa(data.Dislikes),
			},
		})
	}

	if err := cursor.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unkown cursor error: %v", err))
	}
	return nil
}

func (s *CryptoServiceServer) ReadCrypto(ctx context.Context, req *pb.ReadCryptoRequest) (*pb.ReadCryptoResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	result := cryptoDb.FindOne(ctx, bson.M{"_id": objectId})
	data := models.CryptoItem{}
	if err := result.Decode(&data); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find crypto with Object Id %s: %v", req.GetId(), err))
	}

	response := &pb.ReadCryptoResponse{
		Crypto: &pb.Crypto{
			Id:          objectId.Hex(),
			Name:        data.Name,
			Description: data.Description,
			Likes:       strconv.Itoa(data.Likes),
			Dislikes:    strconv.Itoa(data.Dislikes),
		},
	}

	return response, nil
}

func (s *CryptoServiceServer) UpdateCrypto(ctx context.Context, req *pb.UpdateCryptoRequest) (*pb.UpdateCryptoResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Could not convert the supplied crypto id to a MongoDB ObjectId: %v", err),
		)
	}

	update := bson.M{
		"name":        strings.ToUpper(req.GetName()),
		"description": strings.Title(req.GetDescription()),
	}
	filter := bson.M{"_id": objectId}

	result := cryptoDb.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))

	data := models.CryptoItem{}
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	return &pb.UpdateCryptoResponse{
		Success: true,
	}, nil
}

func (s *CryptoServiceServer) DeleteCrypto(ctx context.Context, req *pb.DeleteCryptoRequest) (*pb.DeleteCryptoResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	_, err = cryptoDb.DeleteOne(ctx, bson.M{"_id": objectId})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find/delete crypto with id %s: %v", req.GetId(), err))
	}

	return &pb.DeleteCryptoResponse{
		Success: true,
	}, nil
}

func (s *CryptoServiceServer) AddLike(ctx context.Context, req *pb.AddLikeRequest) (*pb.AddLikeResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	data := models.CryptoItem{}
	result := cryptoDb.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	update := bson.M{"likes": data.Likes + 1}
	filter := bson.M{"_id": objectId}

	result = cryptoDb.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	return &pb.AddLikeResponse{
		Crypto: &pb.Crypto{
			Id:       data.Id.Hex(),
			Name:     data.Name,
			Likes:    strconv.Itoa(data.Likes),
			Dislikes: strconv.Itoa(data.Dislikes),
		},
	}, nil
}

func (s *CryptoServiceServer) RemoveLike(ctx context.Context, req *pb.RemoveLikeRequest) (*pb.RemoveLikeResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	data := models.CryptoItem{}
	result := cryptoDb.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	var likes int
	if data.Likes-1 < 0 {
		likes = 0
	} else {
		likes = data.Likes - 1
	}

	update := bson.M{"likes": likes}
	filter := bson.M{"_id": objectId}

	result = cryptoDb.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	return &pb.RemoveLikeResponse{
		Crypto: &pb.Crypto{
			Id:       data.Id.Hex(),
			Name:     data.Name,
			Likes:    strconv.Itoa(data.Likes),
			Dislikes: strconv.Itoa(data.Dislikes),
		},
	}, nil
}

func (s *CryptoServiceServer) AddDislike(ctx context.Context, req *pb.AddDislikeRequest) (*pb.AddDislikeResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	data := models.CryptoItem{}
	result := cryptoDb.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	update := bson.M{"dislikes": data.Dislikes + 1}
	filter := bson.M{"_id": objectId}

	result = cryptoDb.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	return &pb.AddDislikeResponse{
		Crypto: &pb.Crypto{
			Id:       data.Id.Hex(),
			Name:     data.Name,
			Likes:    strconv.Itoa(data.Likes),
			Dislikes: strconv.Itoa(data.Dislikes),
		},
	}, nil
}

func (s *CryptoServiceServer) RemoveDislike(ctx context.Context, req *pb.RemoveDislikeRequest) (*pb.RemoveDislikeResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	data := models.CryptoItem{}
	result := cryptoDb.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	var dislikes int
	if data.Dislikes-1 < 0 {
		dislikes = 0
	} else {
		dislikes = data.Dislikes - 1
	}

	update := bson.M{"dislikes": dislikes}
	filter := bson.M{"_id": objectId}

	result = cryptoDb.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	return &pb.RemoveDislikeResponse{
		Crypto: &pb.Crypto{
			Id:       data.Id.Hex(),
			Name:     data.Name,
			Likes:    strconv.Itoa(data.Likes),
			Dislikes: strconv.Itoa(data.Dislikes),
		},
	}, nil
}

func (s *CryptoServiceServer) CountVotes(ctx context.Context, req *pb.CountVotesRequest) (*pb.CountVotesResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	var total int

	data := models.CryptoItem{}
	result := cryptoDb.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	total = data.Likes + data.Dislikes

	return &pb.CountVotesResponse{
		Name:  data.Name,
		Total: strconv.Itoa(total),
	}, nil
}
