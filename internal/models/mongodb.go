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
	var orders []*Order
	cursor, err := m.Orders.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	err = cursor.All(context.TODO(), &orders)
	return orders, err
}

func (m *MongoDB) UpdateOrderStatus(orderID primitive.ObjectID, status string) error {
	_, err := m.Orders.UpdateOne(
		context.TODO(),
		bson.M{"_id": orderID},
		bson.M{"$set": bson.M{"status": status}},
	)
	return err
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

func (m *MongoDB) CreatePayment(p Payment) error {
	p.CreatedAt = time.Now()
	if p.ID.IsZero() {
		p.ID = primitive.NewObjectID()
	}
	_, err := m.Payments.InsertOne(context.TODO(), p)
	return err
}

func (m *MongoDB) GetFilteredProducts(search string, category string, city string) ([]*Product, error) {
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
	if city != "" {
		filter["city"] = city
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

func (m *MongoDB) GetTotalRevenue() (float64, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"status": "Completed"}},
		{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$amount"}}},
	}

	cursor, err := m.Payments.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return 0, err
	}

	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil || len(results) == 0 {
		return 0, nil
	}

	return results[0]["total"].(float64), nil
}

func (m *MongoDB) GetOrder(id primitive.ObjectID) (*Order, error) {
	var o Order
	err := m.Orders.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (m *MongoDB) GetPaymentByOrderID(orderID primitive.ObjectID) (*Payment, error) {
	var p Payment
	err := m.Payments.FindOne(context.TODO(), bson.M{"order_id": orderID}).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (m *MongoDB) UpdatePaymentStatus(orderID primitive.ObjectID, status string, method string) error {
	_, err := m.Payments.UpdateOne(
		context.TODO(),
		bson.M{"order_id": orderID},
		bson.M{"$set": bson.M{"status": status, "method": method, "updated_at": time.Now()}},
	)
	return err
}

func (m *MongoDB) GetUniqueCities() ([]string, error) {
	values, err := m.Products.Distinct(context.TODO(), "city", bson.M{})
	if err != nil {
		return nil, err
	}

	var cities []string
	for _, v := range values {
		if s, ok := v.(string); ok && s != "" {
			cities = append(cities, s)
		}
	}
	return cities, nil
}

func (m *MongoDB) DeleteUser(id primitive.ObjectID) error {
	_, err := m.Users.DeleteOne(context.TODO(), bson.M{"_id": id})
	return err
}

func (m *MongoDB) GetAllUsers() ([]*User, error) {
	var users []*User
	cursor, err := m.Users.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	err = cursor.All(context.TODO(), &users)
	return users, err
}
