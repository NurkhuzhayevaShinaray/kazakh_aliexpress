package models

import (
	"database/sql"
)

type Order struct {
	ID        int    `json:"id"`
	ProductID int    `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Status    string `json:"status"`
}

type OrderModel struct {
	DB *sql.DB
}

func (m *OrderModel) Insert(productID int, quantity int) (int, error) {
	query := `INSERT INTO orders (product_id, quantity, status) 
              VALUES ($1, $2, 'pending') RETURNING order_id`

	var id int
	err := m.DB.QueryRow(query, productID, quantity).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *OrderModel) GetAll() ([]*Order, error) {
	query := `SELECT order_id, product_id, quantity, status FROM orders`

	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*Order{}

	for rows.Next() {
		o := &Order{}
		err := rows.Scan(&o.ID, &o.ProductID, &o.Quantity, &o.Status)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
