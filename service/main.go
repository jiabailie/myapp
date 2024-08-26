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
	var items []Item

	rows, err := dbConn.Query(context.Background(), "SELECT id, name, price FROM items ORDER BY id ASC")
	if err != nil {
		message := fmt.Sprintf("Unable to execute the query: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item Item
		err := rows.Scan(&item.Id, &item.Name, &item.Price)
		if err != nil {
			message := fmt.Sprintf("Unable to scan the row: %v", err)
			http.Error(w, message, http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		message := fmt.Sprintf("Unable to iterate over the rows: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		message := fmt.Sprintf("Unable to encode the response: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
	}
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
		message := fmt.Sprintf("Unable to execute the query: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func updateItemHandler(w http.ResponseWriter, r *http.Request) {
	var item Item
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		message := fmt.Sprintf("Invalid request payload: %v", err)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	// Ensure that the ID is provided
	if item.Id == 0 {
		http.Error(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	// Prepare SQL statement for updating the item
	sqlStatement := `
    UPDATE items
    SET name = $1, price = $2
    WHERE id = $3`

	// Execute the update statement
	res, err := dbConn.Exec(context.Background(), sqlStatement, item.Name, item.Price, item.Id)
	if err != nil {
		message := fmt.Sprintf("Unable to execute the query: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Check if the item was actually updated
	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "No item found with the given ID", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Item updated successfully"})
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
	mux.HandleFunc("/api/items", getItemsHandler)          // GET handler
	mux.HandleFunc("/api/items/add", addItemHandler)       // POST handler
	mux.HandleFunc("/api/items/update", updateItemHandler) // PUT handler

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
