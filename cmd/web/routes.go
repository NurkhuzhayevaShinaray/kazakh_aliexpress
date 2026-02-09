package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	dynamic := app.session.LoadAndSave

	mux.Handle("/", dynamic(http.HandlerFunc(app.home)))
	mux.Handle("/catalog", dynamic(http.HandlerFunc(app.catalogPage)))
	mux.Handle("/product", dynamic(http.HandlerFunc(app.showProduct)))
	mux.Handle("/login", dynamic(http.HandlerFunc(app.loginUser)))
	mux.Handle("/register", dynamic(http.HandlerFunc(app.register)))
	mux.Handle("/logout", dynamic(http.HandlerFunc(app.logoutUser)))

	mux.Handle("/cart", dynamic(app.requireAuthentication(http.HandlerFunc(app.showCart))))
	mux.Handle("/cart/add", dynamic(app.requireAuthentication(http.HandlerFunc(app.addToCart))))
	mux.Handle("/cart/remove", dynamic(app.requireAuthentication(http.HandlerFunc(app.removeFromCart))))
	mux.Handle("/order/create-from-cart", dynamic(app.requireAuthentication(http.HandlerFunc(app.createOrderFromCart))))

	mux.Handle("/orders", dynamic(app.requireAuthentication(app.requireRole([]string{"customer", "admin"}, http.HandlerFunc(app.listOrdersPage)))))
	mux.Handle("/order", dynamic(app.requireAuthentication(app.requireRole([]string{"customer", "admin"}, http.HandlerFunc(app.showOrder)))))
	mux.Handle("/order/create", dynamic(app.requireAuthentication(app.requireRole([]string{"customer"}, http.HandlerFunc(app.createOrderFromCart)))))
	mux.Handle("/payment/complete", dynamic(app.requireAuthentication(app.requireRole([]string{"customer"}, http.HandlerFunc(app.completePayment)))))
	mux.Handle("/review/add", dynamic(app.requireAuthentication(app.requireRole([]string{"customer"}, http.HandlerFunc(app.addReview)))))

	mux.Handle("/seller/dashboard", dynamic(app.requireAuthentication(app.requireRole([]string{"seller", "admin"}, http.HandlerFunc(app.sellerDashboard)))))
	mux.Handle("/product/create", dynamic(app.requireAuthentication(app.requireRole([]string{"seller", "admin"}, http.HandlerFunc(app.createProduct)))))
	mux.Handle("/product/delete", dynamic(app.requireAuthentication(app.requireRole([]string{"seller", "admin"}, http.HandlerFunc(app.deleteProduct)))))
	mux.Handle("/product/update", dynamic(app.requireAuthentication(app.requireRole([]string{"seller", "admin"}, http.HandlerFunc(app.updateProductForm)))))
	mux.Handle("/product/update/save", dynamic(app.requireAuthentication(app.requireRole([]string{"seller", "admin"}, http.HandlerFunc(app.updateProduct)))))
	mux.Handle("/category/add", dynamic(app.requireAuthentication(app.requireRole([]string{"seller", "admin"}, http.HandlerFunc(app.addCategory)))))

	mux.Handle("/admin/dashboard", dynamic(app.requireAuthentication(app.requireRole([]string{"admin"}, http.HandlerFunc(app.adminDashboard)))))
	mux.Handle("/admin/users", dynamic(app.requireAuthentication(app.requireRole([]string{"admin"}, http.HandlerFunc(app.listUsers)))))
	mux.Handle("/admin/users/delete", dynamic(app.requireAuthentication(app.requireRole([]string{"admin"}, http.HandlerFunc(app.deleteUser)))))
	mux.Handle("/admin/orders", dynamic(app.requireAuthentication(app.requireRole([]string{"admin"}, http.HandlerFunc(app.adminOrders)))))
	mux.Handle("/admin/orders/update", dynamic(app.requireAuthentication(app.requireRole([]string{"admin"}, http.HandlerFunc(app.updateOrderStatus)))))

	mux.HandleFunc("/api/products", app.apiProducts)
	mux.HandleFunc("/api/orders", app.apiListOrders)

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	return app.recoverPanic(app.logRequest(mux))
}
