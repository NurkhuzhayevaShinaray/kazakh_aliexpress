package main

import (
	"context"
	"html/template"
	"kazakh_aliexpress/internal/models"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type application struct {
	DB            *models.MongoDB
	session       *scs.SessionManager
	orderQueue    chan models.Order
	infoLog       *log.Logger
	errorLog      *log.Logger
	templateCache map[string]*template.Template
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	DB_URL := os.Getenv("DB_URL")
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	session := scs.New()
	session.Lifetime = 12 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = false

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(DB_URL))
	if err != nil {
		errorLog.Fatal(err)
	}

	db := client.Database("kazakh_aliexpress")

	app := &application{
		DB: &models.MongoDB{
			Products:   db.Collection("products"),
			Reviews:    db.Collection("reviews"),
			Users:      db.Collection("users"),
			Orders:     db.Collection("orders"),
			Categories: db.Collection("categories"),
			Payments:   db.Collection("payments"),
		},
		session:       session,
		orderQueue:    make(chan models.Order, 20),
		infoLog:       infoLog,
		errorLog:      errorLog,
		templateCache: templateCache,
	}

	go app.orderWorker()

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Println("Server running on http://localhost:8080")
	errorLog.Fatal(srv.ListenAndServe())
}
