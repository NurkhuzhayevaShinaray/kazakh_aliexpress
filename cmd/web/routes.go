package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/product", app.showProduct)
	mux.HandleFunc("/catalog", app.catalogPage)
	mux.HandleFunc("/orders", app.listOrdersPage)

	mux.HandleFunc("/product/create", app.createProduct)
	mux.HandleFunc("/product/delete", app.deleteProduct)
	mux.HandleFunc("/product/update", app.updateProduct)
	mux.HandleFunc("/order/create", app.createOrder)
	mux.HandleFunc("/review/add", app.addReview)

	mux.HandleFunc("/admin/users", app.requireRole("admin", app.listUsers))
	mux.HandleFunc("/category/add", app.addCategory)

	mux.HandleFunc("/api/products", app.apiProducts)
	mux.HandleFunc("/order", app.showOrder)
	mux.HandleFunc("/payment/complete", app.completePayment)
	mux.HandleFunc("/api/orders", app.apiListOrders)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./ui/static/"))))
	return app.logRequest(app.recoverPanic(mux))
}
