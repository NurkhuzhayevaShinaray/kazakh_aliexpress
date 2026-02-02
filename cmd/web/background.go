package main

import "log"

func (app *application) orderWorker() {
	for order := range app.orderQueue {
		log.Println("Processing order for user:", order.UserID.Hex())
		app.DB.CreateOrder(order)
	}
}
