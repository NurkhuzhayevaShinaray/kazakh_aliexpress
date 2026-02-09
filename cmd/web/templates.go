package main

import (
	"html/template"
	"kazakh_aliexpress/internal/models"
	"path/filepath"
)

type TemplateData struct {
	IsAuthenticated bool
	UserRole        string
	UserName        string
	Products        []*models.Product
	Product         *models.Product
	Reviews         []*models.Review
	Orders          []*models.Order
	Order           *models.Order
	Cart            *models.Cart
	Payment         *models.Payment
	Users           []*models.User
	Categories      []*models.Category
	SearchTerm      string
	CategoryName    string
	TotalRevenue    float64
	TotalOrders     int
	Cities          []string
	CurrentYear     int
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := make(map[string]*template.Template)

	pages, err := filepath.Glob("./ui/html/*.page.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		ts, err := template.ParseFiles("./ui/html/base.layout.tmpl")
		if err != nil {
			return nil, err
		}

		partials, err := filepath.Glob("./ui/html/*.partial.tmpl")
		if err != nil {
			return nil, err
		}

		if len(partials) > 0 {
			ts, err = ts.ParseGlob("./ui/html/*.partial.tmpl")
			if err != nil {
				return nil, err
			}
		}

		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
