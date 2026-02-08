package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"kazakh_aliexpress/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type templateData struct {
	Products     []*models.Product
	Product      *models.Product
	Reviews      []*models.Review
	Orders       []*models.Order
	Order        *models.Order
	Payment      *models.Payment
	Users        []*models.User
	Categories   []*models.Category
	SearchTerm   string
	CategoryName string
	TotalRevenue float64
	TotalOrders  int
	Cities       []string
}

func (app *application) adminDashboard(w http.ResponseWriter, r *http.Request) {
	revenue, _ := app.DB.GetTotalRevenue()
	products, _ := app.DB.GetAllProducts()
	orders, _ := app.DB.GetAllOrders()

	app.render(w, "admin_dashboard.page.tmpl", &templateData{
		TotalRevenue: revenue,
		Products:     products,
		TotalOrders:  len(orders),
	})
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	products, _ := app.DB.GetAllProducts()
	app.render(w, "home.page.tmpl", &templateData{Products: products})
}

func (app *application) showProduct(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		app.notFound(w)
		return
	}

	p, err := app.DB.GetProduct(id)
	if err != nil {
		app.serverError(w, err)
		return
	}

	revs, _ := app.DB.GetReviews(p.ID)

	app.render(w, "show.page.tmpl", &templateData{
		Product: p,
		Reviews: revs,
	})
}

func (app *application) showOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		app.notFound(w)
		return
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		app.notFound(w)
		return
	}

	order, err := app.DB.GetOrder(oid)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	payment, _ := app.DB.GetPaymentByOrderID(oid)

	app.render(w, "order_details.page.tmpl", &templateData{
		Order:   order,
		Payment: payment,
	})
}

func (app *application) addReview(w http.ResponseWriter, r *http.Request) {
	pid, _ := primitive.ObjectIDFromHex(r.FormValue("product_id"))
	user := app.DB.FindOrCreateUser(r.FormValue("email"))

	app.DB.AddReview(models.Review{
		ProductID: pid,
		UserID:    user.ID,
		Rating:    5,
		Comment:   r.FormValue("comment"),
	})

	http.Redirect(w, r, "/product?id="+pid.Hex(), http.StatusSeeOther)
}

func (app *application) apiProducts(w http.ResponseWriter, r *http.Request) {
	products, _ := app.DB.GetAllProducts()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (app *application) apiCreateOrder(w http.ResponseWriter, r *http.Request) {

	var o models.Order

	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {

		app.clientError(w, http.StatusBadRequest)

		return

	}

	if o.ID.IsZero() {

		o.ID = primitive.NewObjectID()

	}

	payment := models.Payment{

		OrderID: o.ID,

		Amount: o.TotalPrice,

		Status: "Pending",

		Method: "API_EXTERNAL",
	}

	if err := app.DB.CreatePayment(payment); err != nil {

		app.serverError(w, err)

		return

	}

	app.orderQueue <- o

	json.NewEncoder(w).Encode(map[string]string{

		"status": "queued",

		"order_id": o.ID.Hex(),
	})

}

func (app *application) render(w http.ResponseWriter, page string, data *templateData) {
	files := []string{
		"./ui/html/base.layout.tmpl",
		"./ui/html/" + page,
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverError(w, fmt.Errorf("could not parse templates: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		app.serverError(w, err)
	}
}

func (app *application) register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		app.render(w, "register.page.tmpl", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		app.serverError(w, err)
		return
	}

	user := models.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         "buyer",
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
		app.render(w, "login.page.tmpl", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	var user models.User
	err := app.DB.Users.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) listUsers(w http.ResponseWriter, r *http.Request) {
	var users []*models.User
	cur, _ := app.DB.Users.Find(context.TODO(), bson.M{})
	cur.All(context.TODO(), &users)
	app.render(w, "users.page.tmpl", &templateData{Users: users})
}

func (app *application) adminProducts(w http.ResponseWriter, r *http.Request) {
	products, _ := app.DB.GetAllProducts()
	app.render(w, "admin_products.page.tmpl", &templateData{Products: products})
}

func (app *application) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := primitive.ObjectIDFromHex(r.FormValue("id"))
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)

	_, err := app.DB.Products.UpdateOne(context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"price": price}},
	)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/catalog", http.StatusSeeOther)
}

func (app *application) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err := app.DB.DeleteProduct(id)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/catalog", http.StatusSeeOther)
}

func (app *application) catalogPage(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	catID := r.URL.Query().Get("category")
	city := r.URL.Query().Get("city")

	products, err := app.DB.GetFilteredProducts(search, catID, city)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if search != "" || catID != "" || city != "" {
		products, err = app.DB.GetFilteredProducts(search, catID, city)
	} else {
		products, err = app.DB.GetAllProducts()
	}

	if err != nil {
		app.serverError(w, err)
		return
	}

	categories, _ := app.DB.GetAllCategories()
	cities, _ := app.DB.GetUniqueCities()

	var selectedCatName string
	for _, c := range categories {
		if c.ID.Hex() == catID {
			selectedCatName = c.Name
			break
		}
	}
	app.render(w, "catalog.page.tmpl", &templateData{
		Products:     products,
		Categories:   categories,
		SearchTerm:   search,
		CategoryName: selectedCatName,
		Cities:       cities,
	})
}

func (app *application) listOrdersPage(w http.ResponseWriter, r *http.Request) {
	orders, err := app.DB.GetAllOrders()
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.render(w, "orders.page.tmpl", &templateData{Orders: orders})
}

func (app *application) createProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	stock, _ := strconv.Atoi(r.FormValue("stock"))
	categoryIDStr := r.FormValue("category_id")
	city := r.FormValue("city")

	var catOID primitive.ObjectID

	if categoryIDStr == "NEW" {
		catOID = primitive.NewObjectID()
		newName := r.FormValue("new_category_name")

		newCat := models.Category{
			ID:   catOID,
			Name: newName,
		}

		_, err := app.DB.Categories.InsertOne(context.TODO(), newCat)
		if err != nil {
			app.serverError(w, err)
			return
		}
	} else {
		catOID, _ = primitive.ObjectIDFromHex(categoryIDStr)
	}

	newProduct := models.Product{
		ID:         primitive.NewObjectID(),
		Name:       r.FormValue("name"),
		Price:      price,
		Stock:      stock,
		CategoryID: catOID,
		City:       city,
	}

	_, err := app.DB.Products.InsertOne(context.TODO(), newProduct)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/catalog", http.StatusSeeOther)
}

func (app *application) apiListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := app.DB.GetAllOrders()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userRole := r.URL.Query().Get("role")

		if userRole != role {
			http.Error(w, "Доступ тыйым салынған (Forbidden)", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (app *application) addCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		if name == "" {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		err := app.DB.AddCategory(name)
		if err != nil {
			app.serverError(w, err)
			return
		}
		http.Redirect(w, r, "/catalog", http.StatusSeeOther)
	}
}

func (app *application) createOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	productIDHex := r.FormValue("product_id")
	qty, _ := strconv.Atoi(r.FormValue("qty"))

	productOID, _ := primitive.ObjectIDFromHex(productIDHex)
	user := app.DB.FindOrCreateUser(email)

	orderID := primitive.NewObjectID()

	product, _ := app.DB.GetProductByOID(productOID)
	totalPrice := product.Price * float64(qty)

	newOrder := models.Order{
		ID:         orderID,
		UserID:     user.ID,
		Status:     "Pending",
		TotalPrice: totalPrice,
		Items: []models.OrderItem{
			{
				ProductID: productOID,
				Quantity:  qty,
				UnitPrice: product.Price,
			},
		},
		CreatedAt: time.Now(),
	}

	payment := models.Payment{
		ID:        primitive.NewObjectID(),
		OrderID:   orderID,
		Amount:    totalPrice,
		Status:    "Pending",
		CreatedAt: time.Now(),
	}

	_, err = app.DB.Orders.InsertOne(r.Context(), newOrder)

	if err != nil {
		app.serverError(w, err)
		return
	}

	err = app.DB.CreatePayment(payment)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.orderQueue <- newOrder

	http.Redirect(w, r, "/order?id="+orderID.Hex(), http.StatusSeeOther)
}

func (app *application) completePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	orderIDHex := r.FormValue("order_id")
	method := r.FormValue("method")
	oid, _ := primitive.ObjectIDFromHex(orderIDHex)

	err := app.DB.UpdatePaymentStatus(oid, "Completed", method)
	if err != nil {
		app.serverError(w, err)
		return
	}

	err = app.DB.UpdateOrderStatus(oid, "Paid")
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/order?id="+orderIDHex, http.StatusSeeOther)
}
