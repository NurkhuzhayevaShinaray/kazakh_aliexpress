package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoDB struct {
	Products *mongo.Collection
	Reviews  *mongo.Collection
	Users    *mongo.Collection
	Orders   *mongo.Collection
}

func (m *MongoDB) GetProduct(id string) (*Product, error) {
	oid, _ := primitive.ObjectIDFromHex(id)
	return m.GetProductByOID(oid)
}

func (m *MongoDB) GetProductByOID(id primitive.ObjectID) (*Product, error) {
	var p Product
	err := m.Products.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&p)
	return &p, err
}

func (m *MongoDB) FindOrCreateUser(email string) *User {
	var u User
	err := m.Users.FindOne(context.TODO(), bson.M{"email": email}).Decode(&u)
	if err == nil {
		return &u
	}
	u = User{Email: email, Role: "buyer"}
	res, _ := m.Users.InsertOne(context.TODO(), u)
	u.ID = res.InsertedID.(primitive.ObjectID)
	return &u
}

func (m *MongoDB) AddReview(r Review) {
	r.CreatedAt = time.Now()
	m.Reviews.InsertOne(context.TODO(), r)
}

func (m *MongoDB) GetReviews(pid primitive.ObjectID) ([]*Review, error) {
	var r []*Review
	cur, _ := m.Reviews.Find(context.TODO(), bson.M{"productid": pid})
	cur.All(context.TODO(), &r)
	return r, nil
}

func (m *MongoDB) CreateOrder(o Order) {
	o.CreatedAt = time.Now()
	m.Orders.InsertOne(context.TODO(), o)
}

func (m *MongoDB) GetAllOrders() ([]*Order, error) {
	var o []*Order
	cur, _ := m.Orders.Find(context.TODO(), bson.M{})
	cur.All(context.TODO(), &o)
	return o, nil
}

func (m *MongoDB) GetAllProducts() ([]*Product, error) {
	var products []*Product
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := m.Products.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &products); err != nil {
		return nil, err
	}
	return products, nil
}

func (m *MongoDB) DeleteProduct(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = m.Products.DeleteOne(context.TODO(), bson.M{"_id": oid})
	return err
}
