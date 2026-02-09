package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"kazakh_aliexpress/internal/models"
)

// --- BASE HELPERS ---

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

// --- AUTH HANDLERS ---

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

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		app.serverError(w, err)
		return
	}

	user := models.User{
		ID:           primitive.NewObjectID(),
		Email:        email,
		PasswordHash: string(hashed),
		Role:         role,
		CreatedAt:    time.Now(),
	}

	_, err = app.DB.Users.InsertOne(context.TODO(), user)
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

	var user models.User
	err := app.DB.Users.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
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

// --- PRODUCT & CATALOG HANDLERS ---

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

// --- CUSTOMER HANDLERS ---

func (app *application) listOrdersPage(w http.ResponseWriter, r *http.Request) {
	userIDHex := app.session.GetString(r.Context(), "authenticatedUserID")
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		app.serverError(w, err)
		return
	}

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
	app.DB.AddReview(models.Review{ProductID: pid, UserID: uid, Rating: 5, Comment: r.FormValue("comment")})
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
	app.DB.Products.InsertOne(context.TODO(), newP)
	http.Redirect(w, r, "/seller/dashboard", http.StatusSeeOther)
}

func (app *application) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := primitive.ObjectIDFromHex(r.FormValue("id"))
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	app.DB.Products.UpdateOne(context.TODO(), bson.M{"_id": id}, bson.M{"$set": bson.M{"price": price}})
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
	users, err := app.DB.GetAllUsers()
	if err != nil {
		app.serverError(w, err)
		return
	}
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

func (app *application) adminProducts(w http.ResponseWriter, r *http.Request) {
	products, _ := app.DB.GetAllProducts()
	app.render(w, r, "admin_dashboard.page.tmpl", &TemplateData{Products: products})
}

func (app *application) apiProducts(w http.ResponseWriter, r *http.Request) {
	p, _ := app.DB.GetAllProducts()
	json.NewEncoder(w).Encode(p)
}

func (app *application) apiListOrders(w http.ResponseWriter, r *http.Request) {
	o, _ := app.DB.GetAllOrders()
	json.NewEncoder(w).Encode(o)
}
