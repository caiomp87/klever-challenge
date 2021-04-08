package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type CryptoItem struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Likes       int                `bson:"likes" json:"likes"`
	Dislikes    int                `bson:"dislikes" json:"dislikes"`
}
