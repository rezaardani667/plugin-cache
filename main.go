package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sidra-gateway/go-pdk/server"
	"golang.org/x/net/context"
)

var redisClient *redis.Client
var ctx = context.Background()
var cacheTTL time.Duration 

//Inisialisasi Redis client dan baca konfigurasi dari environment variables
func init() {
	// Baca alamat Redis dari Env Var atau gunakan default
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Baca password Redis dari Env Var (default kosong)
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// Baca database Redis dari Env Var (default 0)
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB")) 
	if db == 0 {
		db = 0
	}

	// Baca TTL cache dari Env Var (default 5 menit jika tidak ada atau salah format)
	ttl, err := strconv.Atoi(os.Getenv("CACHE_TTL"))
	if err != nil || ttl <= 0 {
		ttl = 300 
	}
	cacheTTL = time.Duration(ttl) * time.Second

	// Inisialisasi Redis client dengan opsi yang sudah diambil dari Env Var
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       db,
	})

	// Cek koneksi ke Redis dan hentikan program jika gagal
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalln("Error connecting to Redis:", err)
	}
	log.Println("Connected to Redis at", redisAddr)
}

// Fungsi untuk menghasilkan cache key unik berdasarkan method, path, dan body request
func generateCacheKey(method, path, body string) string {
	// Buat hash SHA256 dari body request agar key unik untuk setiap konten
	hash := sha256.Sum256([]byte(body))
	bodyHash := hex.EncodeToString(hash[:])
	return fmt.Sprintf("cache:%s:%s:%s", method, path, bodyHash)
}

// Handler untuk menangani fase `access` (cek cache di Redis)
func cacheHandler(req server.Request) server.Response {
	//Generate cache key berdasarkan method, path, dan hash body
	cacheKey := generateCacheKey(req.Method, req.Path, req.Body)

	//Cek apakah data sdh ada di Redis
	cachedData, err := redisClient.Get(ctx, cacheKey).Result()
	
	if err == nil {
		// Cache hit
		ttl, _ := redisClient.TTL(ctx, cacheKey).Result()
		log.Println("Cache hit, returning data from Redis.")
		return server.Response{
			StatusCode: 200,
			Body:       cachedData,
			Headers:    map[string]string{"Cache-Control": fmt.Sprintf("public,max-age=%d", int(ttl.Seconds()))},
		}
	}

	// Buat request ke backend
	backendResponse := fmt.Sprintf("Response from backend for path: %s", req.Path)
	log.Println("Cache miss. Accessing backend...")
	return server.Response{
		StatusCode: 200,
		Body:       backendResponse,
		Headers:    map[string]string{"Cache-Control": "no-cache"},
	}
}

// Handler untuk menangani fase `header_filter` (simpan respons ke cache)
func cacheResponseHandler(req server.Request) server.Response {
	//Generate cache key berdasarkan method, path, dan body request
		cacheKey := generateCacheKey(req.Method, req.Path, req.Body)
		err := redisClient.Set(ctx, cacheKey, req.Body, cacheTTL).Err()
		if err != nil {
			log.Println("Error saving data to Redis:", err)
	} else {
		log.Println("Data saved to Redis with key:", cacheKey)
	}
	return server.Response{}
}

func main() {
	fmt.Println("Memulai plugin cache dengan Env Var...")

	//Start both servers for different phases
	go func() {
		err := server.NewServer("cache.response", cacheResponseHandler).Start() 
		if err != nil {
			fmt.Println("Error starting cache response handler:", err)
		}
	}()
	err := server.NewServer("cache", cacheHandler).Start() 
	if err != nil {
		fmt.Println("Error starting cache handler:", err)
	}
}