package models

import (
	"database/sql"
	"log"
	"time"
)

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}
type ProductModel struct {
	DB *sql.DB
}

func (m *ProductModel) Insert(name string, price float64) (int, error) {
	query := `INSERT INTO products (name, price) VALUES ($1, $2) RETURNING product_id`

	var id int
	err := m.DB.QueryRow(query, name, price).Scan(&id)
	if err != nil {
		return 0, err
	}

	go func(productID int) {
		time.Sleep(500 * time.Millisecond)
		log.Printf("Audit: Background task successfully processed product ID: %d", productID)
	}(id)

	return id, nil
}

func (m *ProductModel) Get(id int) (*Product, error) {
	query := `SELECT product_id, name, price FROM products WHERE product_id = $1`

	var p Product
	err := m.DB.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.Price)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
