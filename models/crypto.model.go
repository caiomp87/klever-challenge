package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CryptoItem struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Likes       int64              `bson:"likes" json:"likes"`
	Dislikes    int64              `bson:"dislikes" json:"dislikes"`
	VoteRate    int64              `bson:"voteRate" json:"voteRate"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}
