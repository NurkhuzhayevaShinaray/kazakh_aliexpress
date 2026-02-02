package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"kazakh_aliexpress/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type templateData struct {
	Products []*models.Product
	Product  *models.Product
	Reviews  []*models.Review
	Orders   []*models.Order
	Users    []*models.User
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

func (app *application) createOrder(w http.ResponseWriter, r *http.Request) {
	pid, _ := primitive.ObjectIDFromHex(r.FormValue("product_id"))
	qty, _ := strconv.Atoi(r.FormValue("qty"))

	product, err := app.DB.GetProductByOID(pid)
	if err != nil {
		app.notFound(w)
		return
	}

	user := app.DB.FindOrCreateUser(r.FormValue("email"))

	order := models.Order{
		UserID: user.ID,
		Status: "Pending",
		Items: []models.OrderItem{
			{
				ProductID: pid,
				Quantity:  qty,
				UnitPrice: product.Price,
			},
		},
		TotalPrice: product.Price * float64(qty),
	}

	app.orderQueue <- order
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) apiProducts(w http.ResponseWriter, r *http.Request) {
	products, _ := app.DB.GetAllProducts()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (app *application) apiCreateOrder(w http.ResponseWriter, r *http.Request) {
	var o models.Order
	json.NewDecoder(r.Body).Decode(&o)
	app.orderQueue <- o
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
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
	var products []*models.Product
	var err error

	if search != "" {
		filter := bson.M{"name": bson.M{"$regex": search, "$options": "i"}}
		cur, _ := app.DB.Products.Find(context.TODO(), filter)
		cur.All(context.TODO(), &products)
	} else {
		products, err = app.DB.GetAllProducts()
	}

	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, "catalog.page.tmpl", &templateData{
		Products: products,
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
	if r.Method == http.MethodPost {
		price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
		stock, _ := strconv.Atoi(r.FormValue("stock"))

		newProduct := models.Product{
			ID:    primitive.NewObjectID(),
			Name:  r.FormValue("name"),
			Price: price,
			Stock: stock,
		}

		_, err := app.DB.Products.InsertOne(context.TODO(), newProduct)
		if err != nil {
			app.serverError(w, err)
			return
		}
		http.Redirect(w, r, "/catalog", http.StatusSeeOther)
	}
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
