package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/jackc/pgx/v4"
	"github.com/rs/cors"
)

var dbConn *pgx.Conn
var redisClient *Redis

const (
	REDIS_TTL = 10 // 10 minutes
)

type Item struct {
	Id    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type Redis struct {
	RedisClient redis.Client
}

func initRedis() (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		// Container name + port since we are using docker
		Addr:     "redis:6379",
		Password: "",
	})

	if client == nil {
		return nil, errors.New("unable to connect to redis")
	}
	return &Redis{RedisClient: *client}, nil
}

func addItemToRedis(item Item) error {
	// add to redis
	cachedKey := strconv.Itoa(item.Id)
	marshalContent, err := json.Marshal(item)
	if err != nil {
		message := fmt.Sprintf("unable to marshal the item: %v", err)
		return errors.New(message)
	}
	cacheErr := redisClient.RedisClient.Set(cachedKey, marshalContent, REDIS_TTL*time.Minute).Err()
	if cacheErr != nil {
		message := fmt.Sprintf("unable to cache the item: %v", cacheErr)
		return errors.New(message)
	}

	// verify if it was cached
	cachedItem, err := redisClient.RedisClient.Get(cachedKey).Bytes()
	if err != nil {
		message := fmt.Sprintf("unable to retrieve the cached item: %v", err)
		return errors.New(message)
	}
	var cachedItemObj Item
	err = json.Unmarshal(cachedItem, &cachedItemObj)
	if err != nil {
		message := fmt.Sprintf("unable to unmarshal the cached item: %v", err)
		return errors.New(message)
	}
	if item.Id != cachedItemObj.Id || item.Name != cachedItemObj.Name || item.Price != cachedItemObj.Price {
		message := "the cached item is different from the original item"
		return errors.New(message)
	}

	return nil
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

	// add to redis
	for _, item := range items {
		err := addItemToRedis(item)
		if err != nil {
			message := fmt.Sprintf("Unable to cache the item: %v", err)
			http.Error(w, message, http.StatusInternalServerError)
			return
		}
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

	// add to database
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

	// add to redis
	item.Id = id
	err = addItemToRedis(item)
	if err != nil {
		message := fmt.Sprintf("Unable to cache the item: %v", err)
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

	// update in redis
	err = addItemToRedis(item)
	if err != nil {
		message := fmt.Sprintf("Unable to cache the item: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
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

func getItemByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the ID from the URL query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Convert the ID to an integer
	_, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	cachedItem, err := redisClient.RedisClient.Get(idStr).Bytes()
	if err != nil {
		message := fmt.Sprintf("unable to retrieve the cached item: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
	var cachedItemObj Item
	err = json.Unmarshal(cachedItem, &cachedItemObj)
	if err != nil {
		message := fmt.Sprintf("unable to unmarshal the cached item: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Return the item as a JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cachedItemObj); err != nil {
		http.Error(w, "Unable to encode response", http.StatusInternalServerError)
	}
}

func main() {
	var err error
	redisClient, err = initRedis()
	if err != nil {
		log.Fatalf("Unable to connect to redis: %v\n", err)
	}

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
	mux.HandleFunc("/api/items/get", getItemByIDHandler)   // GET item by ID handler

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
