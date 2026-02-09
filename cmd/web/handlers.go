package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"kazakh_aliexpress/internal/models"
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

func (app *application) catalogPage(w http.ResponseWriter, r *http.Request) {
	products, _ := app.DB.GetAllProducts()
	categories, _ := app.DB.GetAllCategories()
	cities, _ := app.DB.GetUniqueCities()
	app.render(w, r, "catalog.page.tmpl", &TemplateData{
		Products:   products,
		Categories: categories,
		Cities:     cities,
	})
}

func (app *application) showProduct(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	p, err := app.DB.GetProduct(id)
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}
	revs, _ := app.DB.GetReviews(p.ID)
	app.render(w, r, "show.page.tmpl", &TemplateData{Product: p, Reviews: revs})
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

func (app *application) createOrder(w http.ResponseWriter, r *http.Request) {
	uidHex := app.session.GetString(r.Context(), "authenticatedUserID")
	uid, _ := primitive.ObjectIDFromHex(uidHex)
	pid, _ := primitive.ObjectIDFromHex(r.FormValue("product_id"))

	p, err := app.DB.GetProductByOID(pid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	order := models.Order{
		ID:         primitive.NewObjectID(),
		UserID:     uid,
		Status:     "Pending",
		TotalPrice: p.Price,
		CreatedAt:  time.Now(),
	}

	_, err = app.DB.Orders.InsertOne(r.Context(), order)
	if err != nil {
		app.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
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
	sid, _ := primitive.ObjectIDFromHex(app.session.GetString(r.Context(), "authenticatedUserID"))
	products, _ := app.DB.GetProductsBySeller(sid)
	app.render(w, r, "seller_dashboard.page.tmpl", &TemplateData{Products: products})
}

func (app *application) createProduct(w http.ResponseWriter, r *http.Request) {
	sellerIDHex := app.session.GetString(r.Context(), "authenticatedUserID")
	sellerID, _ := primitive.ObjectIDFromHex(sellerIDHex)
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	catID, _ := primitive.ObjectIDFromHex(r.FormValue("category_id"))

	newP := models.Product{
		ID: primitive.NewObjectID(), Name: r.FormValue("name"), Price: price,
		Stock: 10, City: r.FormValue("city"), CategoryID: catID,
		SellerID: sellerID, Description: r.FormValue("description"),
	}
	app.DB.Products.InsertOne(r.Context(), newP)
	http.Redirect(w, r, "/seller/dashboard", http.StatusSeeOther)
}

func (app *application) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := primitive.ObjectIDFromHex(r.FormValue("id"))
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	app.DB.Products.UpdateOne(r.Context(), bson.M{"_id": id}, bson.M{"$set": bson.M{"price": price}})
	http.Redirect(w, r, "/catalog", http.StatusSeeOther)
}

func (app *application) deleteProduct(w http.ResponseWriter, r *http.Request) {
	app.DB.DeleteProduct(r.FormValue("id"))
	http.Redirect(w, r, "/catalog", http.StatusSeeOther)
}

func (app *application) addCategory(w http.ResponseWriter, r *http.Request) {
	app.DB.AddCategory(r.FormValue("name"))
	http.Redirect(w, r, "/catalog", http.StatusSeeOther)
}

func (app *application) adminDashboard(w http.ResponseWriter, r *http.Request) {
	products, _ := app.DB.GetAllProducts()
	revenue, _ := app.DB.GetTotalRevenue()
	totalOrders, _ := app.DB.GetTotalOrderCount()

	app.render(w, r, "admin_dashboard.page.tmpl", &TemplateData{
		Products:     products,
		TotalRevenue: revenue,
		TotalOrders:  int(totalOrders),
	})
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
	uidHex := app.session.GetString(r.Context(), "authenticatedUserID")
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	cartItems, err := app.DB.GetUserCart(uid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	displayCart := &models.Cart{
		Items:      []models.CartItem{},
		TotalPrice: 0,
	}

	for _, item := range cartItems {
		product, err := app.DB.GetProductByOID(item.ProductID)
		if err != nil {
			continue
		}

		lineTotal := product.Price * float64(item.Quantity)

		displayCart.Items = append(displayCart.Items, models.CartItem{
			ProductID: item.ProductID,
			Name:      product.Name,
			Price:     product.Price,
			Quantity:  item.Quantity,
			Total:     lineTotal,
		})
		displayCart.TotalPrice += lineTotal
	}

	app.render(w, r, "cart.page.tmpl", &TemplateData{
		Cart: displayCart,
	})
}

func (app *application) addToCart(w http.ResponseWriter, r *http.Request) {
	pid, _ := primitive.ObjectIDFromHex(r.FormValue("product_id"))
	uid, _ := primitive.ObjectIDFromHex(app.session.GetString(r.Context(), "authenticatedUserID"))

	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	if qty == 0 {
		qty = 1
	}

	filter := bson.M{"user_id": uid, "product_id": pid}
	update := bson.M{"$inc": bson.M{"quantity": qty}}
	opts := options.Update().SetUpsert(true)

	_, err := app.DB.Users.Database().Collection("cart").UpdateOne(r.Context(), filter, update, opts)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (app *application) createOrderFromCart(w http.ResponseWriter, r *http.Request) {
	uidHex := app.session.GetString(r.Context(), "authenticatedUserID")
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	cartItems, err := app.DB.GetUserCart(uid)
	if err != nil || len(cartItems) == 0 {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	var total float64
	for _, item := range cartItems {
		product, err := app.DB.GetProductByOID(item.ProductID)
		if err == nil {
			total += (product.Price * float64(item.Quantity))
		}
	}

	order := models.Order{
		ID:         primitive.NewObjectID(),
		UserID:     uid,
		TotalPrice: total,
		Status:     "Paid",
		CreatedAt:  time.Now(),
	}

	_, err = app.DB.Orders.InsertOne(r.Context(), order)
	if err != nil {
		app.serverError(w, err)
		return
	}

	_, err = app.DB.Users.Database().Collection("cart").DeleteMany(r.Context(), bson.M{"user_id": uid})
	if err != nil {
		app.errorLog.Printf("Could not clear cart for user %s: %v", uidHex, err)
	}

	app.session.Put(r.Context(), "flash", "Order placed successfully!")
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
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
