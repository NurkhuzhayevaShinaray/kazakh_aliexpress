package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kazakh_aliexpress/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (app *application) addDefaultData(td *TemplateData, r *http.Request) *TemplateData {
	if td == nil {
		td = &TemplateData{}
	}
	td.CurrentYear = time.Now().Year()
	td.IsAuthenticated = app.isAuthenticated(r)

	if td.IsAuthenticated {
		td.UserRole = app.session.GetString(r.Context(), "userRole")
		td.UserName = app.session.GetString(r.Context(), "userEmail")
	}
	return td
}

func (app *application) render(w http.ResponseWriter, r *http.Request, page string, data *TemplateData) {
	ts, ok := app.templateCache[page]
	if !ok {
		app.serverError(w, fmt.Errorf("the template %s does not exist", page))
		return
	}

	buf := new(bytes.Buffer)
	err := ts.ExecuteTemplate(buf, "base", app.addDefaultData(data, r))
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

func (app *application) register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		app.render(w, r, "register.page.tmpl", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	role := r.FormValue("role")
	if role == "" {
		role = "customer"
	}

	err := app.UserRepository.Insert(email, password, role)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *application) loginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		app.render(w, r, "login.page.tmpl", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := app.UserRepository.Authenticate(email, password)
	if err != nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	app.session.Put(r.Context(), "authenticatedUserID", user.ID.Hex())
	app.session.Put(r.Context(), "userRole", user.Role)
	app.session.Put(r.Context(), "userEmail", user.Email)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) logoutUser(w http.ResponseWriter, r *http.Request) {
	app.session.Remove(r.Context(), "authenticatedUserID")
	app.session.Destroy(r.Context())
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	products, err := app.DB.GetAllProducts()
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.render(w, r, "home.page.tmpl", &TemplateData{Products: products})
}

func (app *application) listOrdersPage(w http.ResponseWriter, r *http.Request) {
	userIDHex := app.session.GetString(r.Context(), "authenticatedUserID")
	userID, _ := primitive.ObjectIDFromHex(userIDHex)

	orders, err := app.DB.GetOrdersByUser(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "orders.page.tmpl", &TemplateData{Orders: orders})
}

func (app *application) showOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	oid, _ := primitive.ObjectIDFromHex(id)
	order, _ := app.DB.GetOrder(oid)
	app.render(w, r, "order_details.page.tmpl", &TemplateData{Order: order})
}

func (app *application) completePayment(w http.ResponseWriter, r *http.Request) {
	oid, _ := primitive.ObjectIDFromHex(r.FormValue("order_id"))
	app.DB.UpdateOrderStatus(oid, "Paid")
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

func (app *application) addReview(w http.ResponseWriter, r *http.Request) {
	pid, _ := primitive.ObjectIDFromHex(r.FormValue("product_id"))
	uid, _ := primitive.ObjectIDFromHex(app.session.GetString(r.Context(), "authenticatedUserID"))

	rating, _ := strconv.Atoi(r.FormValue("rating"))

	app.DB.AddReview(models.Review{
		ProductID: pid,
		UserID:    uid,
		Rating:    rating,
		Comment:   r.FormValue("comment")})

	http.Redirect(w, r, "/product?id="+pid.Hex(), http.StatusSeeOther)
}

func (app *application) sellerDashboard(w http.ResponseWriter, r *http.Request) {
	sellerIDHex := app.session.GetString(r.Context(), "authenticatedUserID")
	sellerID, _ := primitive.ObjectIDFromHex(sellerIDHex)

	var products []*models.Product
	cursor, err := app.DB.Products.Find(r.Context(), bson.M{"seller_id": sellerID})
	if err == nil {
		cursor.All(r.Context(), &products)
	}

	categories, err := app.DB.GetAllCategories()
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := &TemplateData{
		Products:   products,
		Categories: categories,
	}

	app.render(w, r, "seller_dashboard.page.tmpl", data)
}

func (app *application) createProduct(w http.ResponseWriter, r *http.Request) {
	sellerIDHex := app.session.GetString(r.Context(), "authenticatedUserID")
	sellerID, _ := primitive.ObjectIDFromHex(sellerIDHex)
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	catIDHex := r.PostFormValue("category_id")

	catID, err := primitive.ObjectIDFromHex(catIDHex)
	if err != nil {
		app.serverError(w, err)
		return
	}

	newP := models.Product{
		ID:          primitive.NewObjectID(),
		Name:        r.FormValue("name"),
		Price:       price,
		Stock:       10,
		City:        r.FormValue("city"),
		CategoryID:  catID,
		SellerID:    sellerID,
		Description: r.FormValue("description"),
	}
	app.DB.Products.InsertOne(r.Context(), newP)
	http.Redirect(w, r, "/seller/dashboard", http.StatusSeeOther)
}

func (app *application) deleteProduct(w http.ResponseWriter, r *http.Request) {
	app.DB.DeleteProduct(r.FormValue("id"))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (app *application) addCategory(w http.ResponseWriter, r *http.Request) {
	app.DB.AddCategory(r.FormValue("name"))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (app *application) adminDashboard(w http.ResponseWriter, r *http.Request) {
	products, err := app.DB.GetAllProducts()
	if err != nil {
		app.serverError(w, err)
		return
	}

	revenue, err := app.DB.GetTotalRevenue()
	if err != nil {
		revenue = 0
	}

	totalOrders, err := app.DB.GetTotalOrderCount()
	if err != nil {
		totalOrders = 0
	}

	data := app.addDefaultData(&TemplateData{
		Products:     products,
		TotalRevenue: revenue,
		TotalOrders:  int(totalOrders),
	}, r)

	app.render(w, r, "admin_dashboard.page.tmpl", data)
}

func (app *application) listUsers(w http.ResponseWriter, r *http.Request) {
	users, _ := app.DB.GetAllUsers()
	app.render(w, r, "admin_users.page.tmpl", &TemplateData{Users: users})
}

func (app *application) deleteUser(w http.ResponseWriter, r *http.Request) {
	oid, _ := primitive.ObjectIDFromHex(r.FormValue("id"))
	app.DB.DeleteUser(oid)
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (app *application) adminOrders(w http.ResponseWriter, r *http.Request) {
	orders, _ := app.DB.GetAllOrders()
	app.render(w, r, "admin_orders.page.tmpl", &TemplateData{Orders: orders})
}

func (app *application) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	oid, _ := primitive.ObjectIDFromHex(r.FormValue("id"))
	app.DB.UpdateOrderStatus(oid, r.FormValue("status"))
	http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
}

func (app *application) apiProducts(w http.ResponseWriter, r *http.Request) {
	p, _ := app.DB.GetAllProducts()
	json.NewEncoder(w).Encode(p)
}

func (app *application) apiListOrders(w http.ResponseWriter, r *http.Request) {
	o, _ := app.DB.GetAllOrders()
	json.NewEncoder(w).Encode(o)
}

func (app *application) showCart(w http.ResponseWriter, r *http.Request) {
	userIDStr := app.session.GetString(r.Context(), "authenticatedUserID")
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	cartItems, err := app.DB.GetUserCart(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	var grandTotal float64
	for _, item := range cartItems {
		item.Total = item.Price * float64(item.Quantity)
		grandTotal += item.Total
	}

	data := app.addDefaultData(&TemplateData{}, r)
	data.Cart = &models.Cart{
		Items:      cartItems,
		TotalPrice: grandTotal,
	}

	app.render(w, r, "cart.page.tmpl", data)
}

func (app *application) removeFromCart(w http.ResponseWriter, r *http.Request) {
	pid, err := primitive.ObjectIDFromHex(r.FormValue("product_id"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	uid, _ := primitive.ObjectIDFromHex(app.session.GetString(r.Context(), "authenticatedUserID"))

	filter := bson.M{"user_id": uid, "product_id": pid}
	_, err = app.DB.Users.Database().Collection("cart").DeleteOne(r.Context(), filter)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (app *application) updateProductForm(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	product, err := app.DB.GetProduct(id)
	if err != nil {
		app.notFound(w)
		return
	}

	categories, _ := app.DB.GetAllCategories()

	app.render(w, r, "update_product.page.tmpl", &TemplateData{
		Product:    product,
		Categories: categories,
	})
}

func (app *application) updateProduct(w http.ResponseWriter, r *http.Request) {
	idHex := r.FormValue("id")
	oid, _ := primitive.ObjectIDFromHex(idHex)
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	catID, _ := primitive.ObjectIDFromHex(r.FormValue("category_id"))

	updatedP := models.Product{
		ID:          oid,
		Name:        r.FormValue("name"),
		Price:       price,
		City:        r.FormValue("city"),
		CategoryID:  catID,
		Description: r.FormValue("description"),
	}

	err := app.DB.UpdateProduct(updatedP)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r.Context(), "flash", "Product updated successfully!")
	http.Redirect(w, r, "/seller/dashboard", http.StatusSeeOther)
}

func (app *application) addToCart(w http.ResponseWriter, r *http.Request) {
	pidHex := r.FormValue("product_id")
	pid, _ := primitive.ObjectIDFromHex(pidHex)
	uid, _ := primitive.ObjectIDFromHex(app.session.GetString(r.Context(), "authenticatedUserID"))

	product, err := app.DB.GetProduct(pidHex)
	if err != nil {
		app.serverError(w, err)
		return
	}

	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	if qty <= 0 {
		qty = 1
	}

	filter := bson.M{"user_id": uid, "product_id": pid}
	update := bson.M{
		"$set": bson.M{
			"name":  product.Name,
			"price": product.Price,
		},
		"$inc": bson.M{"quantity": qty},
	}
	opts := options.Update().SetUpsert(true)

	_, err = app.DB.Users.Database().Collection("cart").UpdateOne(r.Context(), filter, update, opts)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (app *application) createOrderFromCart(w http.ResponseWriter, r *http.Request) {
	userIDStr := app.session.GetString(r.Context(), "authenticatedUserID")
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	paymentMethod := r.FormValue("payment_method")

	cartItems, err := app.DB.GetUserCart(userID)
	if err != nil || len(cartItems) == 0 {
		app.session.Put(r.Context(), "error", "Корзина пуста")
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	var orderItems []models.OrderItem
	var total float64

	for _, item := range cartItems {
		itemTotal := item.Price * float64(item.Quantity)

		orderItems = append(orderItems, models.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.Price,
		})
		total += itemTotal
	}

	order := models.Order{
		ID:            primitive.NewObjectID(),
		UserID:        userID,
		Status:        "Pending",
		TotalPrice:    total,
		PaymentMethod: paymentMethod,
		Items:         orderItems,
		CreatedAt:     time.Now(),
	}

	err = app.DB.CreateOrder(order)
	if err != nil {
		app.serverError(w, err)
		return
	}

	_ = app.DB.ClearCart(userID)

	app.session.Put(r.Context(), "flash", "Заказ успешно оформлен!")
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

func (app *application) catalogPage(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")
	city := r.URL.Query().Get("city")

	products, err := app.DB.GetFilteredProducts(search, category, city)
	if err != nil {
		app.serverError(w, err)
		return
	}

	categories, _ := app.DB.GetAllCategories()
	cities, _ := app.DB.GetUniqueCities()

	app.render(w, r, "catalog.page.tmpl", &TemplateData{
		Products:   products,
		Categories: categories,
		Cities:     cities,
		SearchTerm: search,
	})
}

func (app *application) showProduct(w http.ResponseWriter, r *http.Request) {
	idHex := r.URL.Query().Get("id")

	p, err := app.DB.GetProduct(idHex)
	if err != nil {
		app.notFound(w)
		return
	}

	revs, _ := app.DB.GetReviews(p.ID)

	data := app.addDefaultData(&TemplateData{}, r)
	data.Product = p
	data.Reviews = revs

	app.render(w, r, "show.page.tmpl", data)
}
