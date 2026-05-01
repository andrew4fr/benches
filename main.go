package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
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
	Status    string    `json:"status"` // pending, approved, rejected
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

type AdminAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var db *sql.DB
var adminUsername, adminPassword string
var adminEmail string
var smtpHost, smtpPort, smtpUser, smtpPass string

func main() {
	host := getEnv("PG_HOST", "localhost")
	port := getEnv("PG_PORT", "5432")
	user := getEnv("PG_USER", "postgres")
	password := getEnv("PG_PASSWORD", "postgres")
	dbname := getEnv("PG_DBNAME", "benches")

	adminUsername = getEnv("ADMIN_USERNAME", "admin")
	adminPassword = getEnv("ADMIN_PASSWORD", "admin123")
	adminEmail = getEnv("ADMIN_EMAIL", "admin@example.com")

	smtpHost = getEnv("SMTP_HOST", "")
	smtpPort = getEnv("SMTP_PORT", "587")
	smtpUser = getEnv("SMTP_USER", "")
	smtpPass = getEnv("SMTP_PASS", "")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	log.Println("Успешное подключение к PostgreSQL")

	if err = createTable(); err != nil {
		log.Fatalf("Ошибка создания таблицы: %v", err)
	}

	r := mux.NewRouter()

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

	r.HandleFunc("/api/admin/login", adminLogin).Methods("POST")
	r.HandleFunc("/api/admin/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("OPTIONS")
	r.HandleFunc("/api/admin/pending", getPendingBenches).Methods("GET")
	r.HandleFunc("/api/admin/bench/{id}/approve", approveBench).Methods("POST")
	r.HandleFunc("/api/admin/bench/{id}/reject", rejectBench).Methods("POST")

	r.PathPrefix("/admin/").Handler(http.StripPrefix("/admin/", http.FileServer(http.Dir("./admin"))))

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
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	return err
}

func getBenches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query(`
		SELECT id, name, comment, email, latitude, longitude, photo, status, created_at
		FROM benches
		WHERE status = 'approved'
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
		var status string
		err := rows.Scan(&bench.ID, &bench.Name, &bench.Comment, &bench.Email,
			&bench.Latitude, &bench.Longitude, &photo, &status, &bench.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		bench.Status = status
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

func adminLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var auth AdminAuth
	if err := json.NewDecoder(r.Body).Decode(&auth); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if auth.Username == adminUsername && auth.Password == adminPassword {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"token": "admin-token"})
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "Неверные учётные данные"})
}

func getPendingBenches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query(`
		SELECT id, name, comment, email, latitude, longitude, photo, created_at
		FROM benches
		WHERE status = 'pending'
		ORDER BY created_at ASC
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

		bench.Status = "pending"
		benches = append(benches, bench)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(benches)
}

func approveBench(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	var benchName, userEmail string
	err := db.QueryRow("SELECT name, email FROM benches WHERE id = $1", id).Scan(&benchName, &userEmail)
	if err != nil {
		http.Error(w, "Скамейка не найдена", http.StatusNotFound)
		return
	}

	_, err = db.Exec("UPDATE benches SET status = 'approved' WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go sendStatusEmail(userEmail, benchName, "approved")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Скамейка одобрена"})
}

func rejectBench(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	var benchName, userEmail string
	err := db.QueryRow("SELECT name, email FROM benches WHERE id = $1", id).Scan(&benchName, &userEmail)
	if err != nil {
		http.Error(w, "Скамейка не найдена", http.StatusNotFound)
		return
	}

	_, err = db.Exec("UPDATE benches SET status = 'rejected' WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go sendStatusEmail(userEmail, benchName, "rejected")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Скамейка отклонена"})
}

func sendStatusEmail(toEmail, benchName, status string) {
	if smtpHost == "" {
		log.Printf("SMTP не настроен. Уведомление не отправлено. Скамейка #%s (%s) %s", benchName, toEmail, status)
		return
	}

	subject := ""
	body := ""

	if status == "approved" {
		subject = "Ваша скамейка одобрена! 🪑"
		body = fmt.Sprintf(`
Здравствуйте!

Ваша заявка на добавление скамейки "%s" была одобрена и теперь отображается на карте.

Спасибо за вклад в наше сообщество!

С уважением,
Команда "Найти скамейку рядом"
		`, benchName)
	} else {
		subject = "Статус вашей заявки на скамейку"
		body = fmt.Sprintf(`
Здравствуйте!

К сожалению, ваша заявка на добавление скамейки "%s" была отклонена модератором.

Причины могут быть разные: неточное местоположение, дубликат или несоответствие требованиям.
Вы можете попробовать добавить скамейку снова, исправив возможные ошибки.

С уважением,
Команда "Найти скамейку рядом"
		`, benchName)
	}

	message := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		adminEmail, toEmail, subject, body))

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, adminEmail, []string{toEmail}, message)
	if err != nil {
		log.Printf("Ошибка отправки email: %v", err)
		return
	}

	log.Printf("Email отправлен: %s -> %s (статус: %s)", adminEmail, toEmail, status)
}
