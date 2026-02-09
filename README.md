Kazakh@Express

Kazakh@Express is a localized marketplace platform built with Go and MongoDB, tailored for the Kazakhstan market. It focuses on high performance and clean architecture.
 

-Key Features

Customer: Browse products, manage cart, and track order history.


Seller: Manage inventory via a personal CRUD-enabled dashboard.


Admin: View platform analytics and moderate users/orders.


Auth: Secure registration and login for all user roles.

-Tech Stack

Backend: Go (Golang).


Database: MongoDB (BSON).


Frontend: Go html/template tags.


Security: Role-based Middleware and bcrypt hashing.

-Project Structure

cmd/web: Application entry, routing, and handlers.


internal: Business logic, models, and database repositories.


ui: HTML templates (pages/layouts) and static assets.

-Quick Start

Clone: git clone <repo-url>

Setup: Configure MongoDB connection string in main.go.

Run: go run ./cmd/web

