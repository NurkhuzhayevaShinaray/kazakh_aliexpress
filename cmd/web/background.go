package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

func (app *application) orderWorker() {
	for order := range app.orderQueue {
		for _, item := range order.Items {
			filter := bson.M{"_id": item.ProductID}
			update := bson.M{"$inc": bson.M{"stock": -item.Quantity}}
			_, err := app.DB.Products.UpdateOne(context.TODO(), filter, update)
			if err != nil {
				app.errorLog.Println("Failed to update stock:", err)
			}
		}
		app.DB.CreateOrder(order)
	}
}
