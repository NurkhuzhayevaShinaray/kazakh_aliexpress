package models

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CartItem struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"user_id"`
	ProductID primitive.ObjectID `bson:"product_id"`
	Quantity  int                `bson:"quantity"`
	Name      string
	Price     float64
	Total     float64
}

func (m *MongoDB) GetUserCart(userID primitive.ObjectID) ([]*CartItem, error) {
	var items []*CartItem
	cursor, err := m.Users.Database().Collection("cart").Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	err = cursor.All(context.TODO(), &items)
	return items, err
}
