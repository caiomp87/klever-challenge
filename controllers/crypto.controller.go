package controllers

import (
	"api/app/pb"
	"api/models"
	"context"
	"fmt"
	"strings"
	"time"

	bson "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CryptoServiceServer struct {
	Db  *mongo.Collection
	Ctx context.Context
	pb.UnimplementedCryptoServiceServer
}

func (s *CryptoServiceServer) CreateCrypto(ctx context.Context, req *pb.CreateCryptoRequest) (*pb.CreateCryptoResponse, error) {
	data := models.CryptoItem{
		Name:        strings.ToUpper(req.GetName()),
		Description: strings.Title(req.GetDescription()),
		Likes:       0,
		Dislikes:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result, err := s.Db.InsertOne(s.Ctx, data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}

	data.Id = result.InsertedID.(bson.ObjectID)

	return &pb.CreateCryptoResponse{
		Success: true,
		Crypto: &pb.Crypto{
			Name:        data.Name,
			Description: data.Description,
			Likes:       data.Likes,
			Dislikes:    data.Dislikes,
		},
	}, nil
}

func (s *CryptoServiceServer) ListCryptos(req *pb.ListCryptosRequest, stream pb.CryptoService_ListCryptosServer) error {
	cursor, err := s.Db.Find(s.Ctx, bson.M{}, options.Find().SetSort(bson.M{"voteRate": -1}))
	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}

	defer cursor.Close(s.Ctx)

	var data models.CryptoItem
	for cursor.Next(s.Ctx) {
		err := cursor.Decode(&data)
		if err != nil {
			return status.Errorf(codes.Unavailable, fmt.Sprintf("Could not decode data: %v", err))
		}

		stream.Send(&pb.ListCryptosResponse{
			Crypto: &pb.Crypto{
				Id:          data.Id.Hex(),
				Name:        data.Name,
				Description: data.Description,
				Likes:       data.Likes,
				Dislikes:    data.Dislikes,
			},
		})
	}

	if err := cursor.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unkown cursor error: %v", err))
	}
	return nil
}

func (s *CryptoServiceServer) ReadCrypto(ctx context.Context, req *pb.ReadCryptoRequest) (*pb.ReadCryptoResponse, error) {
	objectId, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	result := s.Db.FindOne(ctx, bson.M{"_id": objectId})

	var data models.CryptoItem
	if err := result.Decode(&data); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find crypto with Object Id %s: %v", req.GetId(), err))
	}

	response := &pb.ReadCryptoResponse{
		Crypto: &pb.Crypto{
			Id:          objectId.Hex(),
			Name:        data.Name,
			Description: data.Description,
			Likes:       data.Likes,
			Dislikes:    data.Dislikes,
		},
	}

	return response, nil
}

func (s *CryptoServiceServer) UpdateCrypto(ctx context.Context, req *pb.UpdateCryptoRequest) (*pb.UpdateCryptoResponse, error) {
	objectId, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Could not convert the supplied crypto id to a MongoDB ObjectId: %v", err),
		)
	}

	update := bson.M{
		"name":        strings.ToUpper(req.GetName()),
		"description": strings.Title(req.GetDescription()),
		"updatedAt":   time.Now(),
	}
	filter := bson.M{"_id": objectId}

	result := s.Db.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))

	var data models.CryptoItem
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	return &pb.UpdateCryptoResponse{
		Success: true,
		Crypto: &pb.Crypto{
			Name:        data.Name,
			Description: data.Description,
			Likes:       data.Likes,
			Dislikes:    data.Dislikes,
		},
	}, nil
}

func (s *CryptoServiceServer) DeleteCrypto(ctx context.Context, req *pb.DeleteCryptoRequest) (*pb.DeleteCryptoResponse, error) {
	objectId, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	_, err = s.Db.DeleteOne(ctx, bson.M{"_id": objectId})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find/delete crypto with id %s: %v", req.GetId(), err))
	}

	return &pb.DeleteCryptoResponse{
		Success: true,
	}, nil
}

func (s *CryptoServiceServer) AddLike(ctx context.Context, req *pb.AddLikeRequest) (*pb.AddLikeResponse, error) {
	objectId, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	var data models.CryptoItem
	result := s.Db.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	update := bson.M{
		"likes":     data.Likes + 1,
		"voteRate":  (data.Likes + 1) - data.Dislikes,
		"updatedAt": time.Now(),
	}
	filter := bson.M{"_id": objectId}

	result = s.Db.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
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
			Likes:    data.Likes,
			Dislikes: data.Dislikes,
		},
	}, nil
}

func (s *CryptoServiceServer) RemoveLike(ctx context.Context, req *pb.RemoveLikeRequest) (*pb.RemoveLikeResponse, error) {
	objectId, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	var data models.CryptoItem
	result := s.Db.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	var likes int64
	if data.Likes-1 < 0 {
		likes = 0
	} else {
		likes = data.Likes - 1
	}

	update := bson.M{
		"likes":     likes,
		"voteRate":  likes - data.Dislikes,
		"updatedAt": time.Now(),
	}
	filter := bson.M{"_id": objectId}

	result = s.Db.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
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
			Likes:    data.Likes,
			Dislikes: data.Dislikes,
		},
	}, nil
}

func (s *CryptoServiceServer) AddDislike(ctx context.Context, req *pb.AddDislikeRequest) (*pb.AddDislikeResponse, error) {
	objectId, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	var data models.CryptoItem
	result := s.Db.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	update := bson.M{
		"dislikes":  data.Dislikes + 1,
		"voteRate":  data.Likes - (data.Dislikes + 1),
		"updatedAt": time.Now(),
	}
	filter := bson.M{"_id": objectId}

	result = s.Db.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
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
			Likes:    data.Likes,
			Dislikes: data.Dislikes,
		},
	}, nil
}

func (s *CryptoServiceServer) RemoveDislike(ctx context.Context, req *pb.RemoveDislikeRequest) (*pb.RemoveDislikeResponse, error) {
	objectId, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	var data models.CryptoItem
	result := s.Db.FindOne(ctx, bson.M{"_id": objectId})
	err = result.Decode(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find crypto with supplied ID: %v", err),
		)
	}

	var dislikes int64
	if data.Dislikes-1 < 0 {
		dislikes = 0
	} else {
		dislikes = data.Dislikes - 1
	}

	update := bson.M{
		"dislikes":  dislikes,
		"voteRate":  data.Likes - dislikes,
		"updatedAt": time.Now(),
	}
	filter := bson.M{"_id": objectId}

	result = s.Db.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
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
			Likes:    data.Likes,
			Dislikes: data.Dislikes,
		},
	}, nil
}

func (s *CryptoServiceServer) CountVotes(ctx context.Context, req *pb.CountVotesRequest) (*pb.CountVotesResponse, error) {
	objectId, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}

	var total int64

	var data models.CryptoItem
	result := s.Db.FindOne(ctx, bson.M{"_id": objectId})
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
		Total: total,
	}, nil
}

func (s *CryptoServiceServer) FilterByName(req *pb.FilterByNameRequest, stream pb.CryptoService_FilterByNameServer) error {
	cursor, err := s.Db.Find(s.Ctx, bson.M{"name": bson.Regex{Pattern: req.GetName(), Options: "i"}}, options.Find().SetSort(bson.M{"likes": -1}))
	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}

	defer cursor.Close(s.Ctx)

	var data models.CryptoItem
	for cursor.Next(s.Ctx) {
		err := cursor.Decode(&data)
		if err != nil {
			return status.Errorf(codes.Unavailable, fmt.Sprintf("Could not decode data: %v", err))
		}

		stream.Send(
			&pb.Crypto{
				Id:          data.Id.Hex(),
				Name:        data.Name,
				Description: data.Description,
				Likes:       data.Likes,
				Dislikes:    data.Dislikes,
			},
		)
	}

	if err := cursor.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unkown cursor error: %v", err))
	}

	return nil
}
