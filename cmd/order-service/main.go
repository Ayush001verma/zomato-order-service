// package main

// import (
// 	"encoding/json"
// 	"log"
// 	"net/http"
// )

// func main() {
// 	mux := http.NewServeMux()

// 	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
// 	})

// 	addr := ":8080"
// 	log.Printf("order-service listening on %s", addr)

// 	if err := http.ListenAndServe(addr, mux); err != nil {
// 		log.Fatal(err)
// 	}
// }

// updated

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Ayush001verma/zomato-order-service/internal/handler"
	"github.com/Ayush001verma/zomato-order-service/internal/repository"
	"github.com/Ayush001verma/zomato-order-service/internal/service"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	ctx := context.Background()

	databaseURL := getEnv("DATABASE_URL",
		"postgres://zomato:zomato123@localhost:5432/orders_db?sslmode=disable")

	pool, err := repository.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("connected to postgres")

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	rdb, err := repository.NewRedisClient(ctx, redisAddr)
	if err != nil {
		log.Printf("WARNING: redis unavailable, running without cache: %v", err)
	} else {
		log.Println("connected to redis")
	}
	defer rdb.Close()

	mux := http.NewServeMux()

	// Wire the layers together: pool -> repository -> service -> handler.
	orderRepo := repository.NewOrderRepository(pool)
	orderSvc := service.NewOrderService(orderRepo)
	orderHandler := handler.NewOrderHandler(orderSvc)
	orderHandler.RegisterRoutes(mux)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(pingCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "db": "down"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "db": "up"})
	})

	addr := ":8080"
	log.Printf("order-service listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
