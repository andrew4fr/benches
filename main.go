package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Bench struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Comment   string    `json:"comment,omitempty"`
	Email     string    `json:"email"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Photo     *string   `json:"photo,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateBenchRequest struct {
	Name      string  `json:"name"`
	Comment   string  `json:"comment,omitempty"`
	Email     string  `json:"email"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Photo     *string `json:"photo,omitempty"`
}

var db *sql.DB

func main() {
	// Получение DSN из переменных окружения
	host := getEnv("PG_HOST", "localhost")
	port := getEnv("PG_PORT", "5432")
	user := getEnv("PG_USER", "postgres")
	password := getEnv("PG_PASSWORD", "postgres")
	dbname := getEnv("PG_DBNAME", "benches")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	// Проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	log.Println("Успешное подключение к PostgreSQL")

	// Создание таблицы
	if err = createTable(); err != nil {
		log.Fatalf("Ошибка создания таблицы: %v", err)
	}

	// Настройка роутера
	r := mux.NewRouter()

	// CORS middleware (ДО маршрутов!)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/api/benches", getBenches).Methods("GET")
	r.HandleFunc("/api/benches", createBench).Methods("POST")
	r.HandleFunc("/api/benches", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("OPTIONS")

	portNum := getEnv("PORT", "8080")
	log.Printf("Сервер запущен на порту %s", portNum)

	if err := http.ListenAndServe(":"+portNum, r); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS benches (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		comment TEXT,
		email VARCHAR(255),
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		photo TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	return err
}

func getBenches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query(`
		SELECT id, name, comment, email, latitude, longitude, photo, created_at
		FROM benches
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var benches []Bench
	for rows.Next() {
		var bench Bench
		var photo sql.NullString
		err := rows.Scan(&bench.ID, &bench.Name, &bench.Comment, &bench.Email,
			&bench.Latitude, &bench.Longitude, &photo, &bench.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if photo.Valid {
			bench.Photo = &photo.String
		}

		benches = append(benches, bench)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(benches)
}

func createBench(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CreateBenchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Name == "" || req.Latitude == 0 || req.Longitude == 0 {
		http.Error(w, "Name, latitude и longitude обязательны", http.StatusBadRequest)
		return
	}

	var benchID int
	var photo *string

	if req.Photo != nil && *req.Photo != "" {
		photo = req.Photo
	}

	err := db.QueryRow(`
		INSERT INTO benches (name, comment, email, latitude, longitude, photo)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.Name, req.Comment, req.Email, req.Latitude, req.Longitude, photo).Scan(&benchID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bench := Bench{
		ID:        benchID,
		Name:      req.Name,
		Comment:   req.Comment,
		Email:     req.Email,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Photo:     photo,
		CreatedAt: time.Now(),
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bench)
}
