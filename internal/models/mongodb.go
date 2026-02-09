package models

import (
	"context"
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
	Carts      *mongo.Collection
}

func (m *MongoDB) GetProductByOID(id primitive.ObjectID) (*Product, error) {
	var p Product
	err := m.Products.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&p)
	return &p, err
}

func (m *MongoDB) GetProduct(id string) (*Product, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return m.GetProductByOID(oid)
}

func (m *MongoDB) GetAllProducts() ([]*Product, error) {
	var products []*Product
	cur, err := m.Products.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	err = cur.All(context.TODO(), &products)
	return products, err
}

func (m *MongoDB) GetProductsBySeller(sellerID primitive.ObjectID) ([]*Product, error) {
	var products []*Product
	cur, err := m.Products.Find(context.TODO(), bson.M{"seller_id": sellerID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	err = cur.All(context.TODO(), &products)
	return products, err
}

func (m *MongoDB) DeleteProduct(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = m.Products.DeleteOne(context.TODO(), bson.M{"_id": oid})
	return err
}

func (m *MongoDB) CreateOrder(o Order) error {
	o.CreatedAt = time.Now()
	o.Status = "Pending"

	for _, item := range o.Items {
		m.Products.UpdateOne(context.TODO(), bson.M{"_id": item.ProductID}, bson.M{"$inc": bson.M{"stock": -item.Quantity}})
	}

	_, err := m.Orders.InsertOne(context.TODO(), o)
	return err
}

func (m *MongoDB) GetCartByUserID(userID primitive.ObjectID) (*Cart, error) {
	var c Cart
	collection := m.Users.Database().Collection("cart")
	err := collection.FindOne(context.TODO(), bson.M{"user_id": userID}).Decode(&c)
	return &c, err
}

func (m *MongoDB) GetOrder(id primitive.ObjectID) (*Order, error) {
	var o Order
	err := m.Orders.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&o)
	return &o, err
}

func (m *MongoDB) GetAllOrders() ([]*Order, error) {
	var orders []*Order
	cur, err := m.Orders.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	err = cur.All(context.TODO(), &orders)
	return orders, err
}

func (m *MongoDB) UpdateOrderStatus(orderID primitive.ObjectID, status string) error {
	_, err := m.Orders.UpdateOne(context.TODO(), bson.M{"_id": orderID}, bson.M{"$set": bson.M{"status": status}})
	return err
}

func (m *MongoDB) GetTotalRevenue() (float64, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"status": bson.M{"$in": []string{"Completed", "Paid"}}}},
		{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$amount"}}},
	}
	cur, err := m.Payments.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return 0, err
	}
	defer cur.Close(context.TODO())
	var results []bson.M
	if err = cur.All(context.TODO(), &results); err != nil || len(results) == 0 {
		return 0, nil
	}
	switch v := results[0]["total"].(type) {
	case float64:
		return v, nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, nil
	}
}

func (m *MongoDB) GetTotalOrderCount() (int64, error) {
	return m.Orders.CountDocuments(context.TODO(), bson.M{})
}

func (m *MongoDB) AddReview(r Review) error {
	r.CreatedAt = time.Now()
	_, err := m.Reviews.InsertOne(context.TODO(), r)
	return err
}

func (m *MongoDB) GetReviews(pid primitive.ObjectID) ([]*Review, error) {
	var reviews []*Review
	cur, err := m.Reviews.Find(context.TODO(), bson.M{"productid": pid})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	err = cur.All(context.TODO(), &reviews)
	return reviews, err
}

func (m *MongoDB) GetAllUsers() ([]*User, error) {
	var users []*User
	cur, err := m.Users.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	err = cur.All(context.TODO(), &users)
	return users, err
}

func (m *MongoDB) DeleteUser(id primitive.ObjectID) error {
	_, err := m.Users.DeleteOne(context.TODO(), bson.M{"_id": id})
	return err
}

func (m *MongoDB) AddCategory(name string) error {
	cat := Category{ID: primitive.NewObjectID(), Name: name}
	_, err := m.Categories.InsertOne(context.TODO(), cat)
	return err
}

func (m *MongoDB) GetAllCategories() ([]*Category, error) {
	var cats []*Category
	cur, err := m.Categories.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	err = cur.All(context.TODO(), &cats)
	return cats, err
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

func (m *MongoDB) UpdateProduct(p Product) error {
	filter := bson.M{"_id": p.ID}
	update := bson.M{
		"$set": bson.M{
			"name":        p.Name,
			"price":       p.Price,
			"city":        p.City,
			"description": p.Description,
			"category_id": p.CategoryID,
		},
	}
	_, err := m.Products.UpdateOne(context.TODO(), filter, update)
	return err
}

func (m *MongoDB) ClearCart(userID primitive.ObjectID) error {
	collection := m.Users.Database().Collection("cart")
	_, err := collection.DeleteOne(context.TODO(), bson.M{"user_id": userID})
	return err
}

func (m *MongoDB) GetOrdersByUser(userID primitive.ObjectID) ([]*Order, error) {
	var orders []*Order
	cur, err := m.Orders.Find(context.TODO(), bson.M{"userid": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	err = cur.All(context.TODO(), &orders)
	return orders, err
}

func (m *MongoDB) GetFilteredProducts(search, category, city string) ([]*Product, error) {
	filter := bson.M{}

	if search != "" {
		filter["name"] = bson.M{"$regex": search, "$options": "i"}
	}

	if category != "" {
		if oid, err := primitive.ObjectIDFromHex(category); err == nil {
			filter["category_id"] = oid
		}
	}

	if city != "" {
		filter["city"] = city
	}

	var products []*Product
	cur, err := m.Products.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	err = cur.All(context.TODO(), &products)
	return products, err
}
