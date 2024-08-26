package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4"
	"github.com/rs/cors"
)

var dbConn *pgx.Conn

type Item struct {
	Id    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func getItemsHandler(w http.ResponseWriter, r *http.Request) {
	items := []Item{
		{Id: 1, Name: "Apple", Price: 0.5},
		{Id: 2, Name: "Banana", Price: 0.2},
		{Id: 3, Name: "Orange", Price: 0.7},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func addItemHandler(w http.ResponseWriter, r *http.Request) {
	var item Item
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		message := fmt.Sprintf("Invalid request payload: %v", err)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	sqlStatement := `
    INSERT INTO items (name, price)
    VALUES ($1, $2)
    RETURNING id`
	id := 0
	err = dbConn.QueryRow(context.Background(), sqlStatement, item.Name, item.Price).Scan(&id)
	if err != nil {
		http.Error(w, "Unable to execute the query", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func main() {
	var err error
	dbConn, err = pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbConn.Close(context.Background())

	log.Println("Connected to PostgreSQL!")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/items", getItemsHandler)    // GET handler
	mux.HandleFunc("/api/items/add", addItemHandler) // POST handler

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	// Use the CORS middleware
	handler := c.Handler(mux)

	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("could not start server: %s\n", err)
	}
}
