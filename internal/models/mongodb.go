package models

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoDB struct {
	Products   *mongo.Collection
	Reviews    *mongo.Collection
	Users      *mongo.Collection
	Orders     *mongo.Collection
	Categories *mongo.Collection
	Payments   *mongo.Collection
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

func (m *MongoDB) CreateOrder(o Order) error {
	o.CreatedAt = time.Now()
	o.Status = "Pending"

	var total float64 = 0
	for _, item := range o.Items {
		product, err := m.GetProductByOID(item.ProductID)
		if err != nil {
			return err
		}

		if product.Stock < item.Quantity {
			return fmt.Errorf("insufficient stock for product: %s", product.Name)
		}

		total += item.UnitPrice * float64(item.Quantity)

		_, err = m.Products.UpdateOne(
			context.TODO(),
			bson.M{"_id": item.ProductID},
			bson.M{"$inc": bson.M{"stock": -item.Quantity}},
		)
	}
	o.TotalPrice = total

	_, err := m.Orders.InsertOne(context.TODO(), o)
	return err
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

func (m *MongoDB) AddCategory(name string) error {
	cat := Category{
		ID:   primitive.NewObjectID(),
		Name: name,
	}
	_, err := m.Categories.InsertOne(context.TODO(), cat)
	return err
}

func (m *MongoDB) GetAllCategories() ([]*Category, error) {
	var cats []*Category
	cur, err := m.Categories.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	err = cur.All(context.TODO(), &cats)
	return cats, err
}

func (m *MongoDB) CreatePayment(orderID primitive.ObjectID, amount float64, method string) (primitive.ObjectID, error) {
	payment := Payment{
		ID:        primitive.NewObjectID(),
		OrderID:   orderID,
		Amount:    amount,
		Status:    "Pending",
		Method:    method,
		CreatedAt: time.Now(),
	}
	res, err := m.Payments.InsertOne(context.TODO(), payment)
	return res.InsertedID.(primitive.ObjectID), err
}

func (m *MongoDB) GetFilteredProducts(search string, category string) ([]*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{}

	if search != "" {
		filter["name"] = bson.M{"$regex": search, "$options": "i"}
	}

	if category != "" {
		oid, err := primitive.ObjectIDFromHex(category)
		if err == nil {
			filter["category_id"] = oid
		}
	}

	var products []*Product
	cursor, err := m.Products.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}
