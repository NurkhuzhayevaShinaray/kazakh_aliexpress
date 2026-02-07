package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Email        string             `bson:"email"`
	PasswordHash string             `bson:"password_hash"`
	Role         string             `bson:"role"`
	CreatedAt    time.Time          `bson:"created_at"`
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

type Category struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name string             `bson:"name" json:"name"`
}

type Payment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID   primitive.ObjectID `bson:"order_id" json:"order_id"`
	Amount    float64            `bson:"amount" json:"amount"`
	Status    string             `bson:"status" json:"status"`
	Method    string             `bson:"method" json:"method"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type Product struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Name       string             `bson:"name"`
	Price      float64            `bson:"price"`
	Stock      int                `bson:"stock"`
	CategoryID primitive.ObjectID `bson:"category_id,omitempty"`
}
