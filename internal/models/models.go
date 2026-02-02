package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Email        string             `bson:"email"`
	PasswordHash string             `bson:"password_hash"` // Standardize to this
	Role         string             `bson:"role"`
	CreatedAt    time.Time          `bson:"created_at"`
}

type Product struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Name  string
	Price float64
	Stock int
}

type Review struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	ProductID primitive.ObjectID
	UserID    primitive.ObjectID
	Rating    int
	Comment   string
	CreatedAt time.Time
}

type Order struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	UserID     primitive.ObjectID
	Status     string
	TotalPrice float64
	Items      []OrderItem
	CreatedAt  time.Time
}

type OrderItem struct {
	ProductID primitive.ObjectID `bson:"product_id"`
	Quantity  int                `bson:"quantity"`
	UnitPrice float64            `bson:"unit_price"`
}
