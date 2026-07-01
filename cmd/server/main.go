package main

import (
	"context"
	"log"
	"net/http"

	"freshtrack/internal/config"
	"freshtrack/internal/db"
	"freshtrack/internal/queue"
	"freshtrack/internal/redisclient"
	"freshtrack/internal/router"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	rdb := redisclient.NewClient(cfg.RedisAddr)
	if err := redisclient.Ping(ctx, rdb); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	go queue.StartScanWorker(ctx, rdb, pool)

	h := router.New(cfg, pool, rdb)

	log.Printf("FreshTrack server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, h); err != nil {
		log.Fatal(err)
	}
}
