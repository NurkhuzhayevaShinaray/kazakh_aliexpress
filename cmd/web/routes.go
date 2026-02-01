package main

import "net/http"

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/product", app.showProduct)
	mux.HandleFunc("/product/view/{id}", app.showProduct)
	mux.HandleFunc("/orders", app.listOrders)
	mux.HandleFunc("/pruducts", app.listProducts)

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	return mux

}
