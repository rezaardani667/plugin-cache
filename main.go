package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sidra-gateway/go-pdk/server"
	"golang.org/x/net/context"
)

var redisClient *redis.Client
var ctx = context.Background()

//Inisialisasi Redis client
func init() {
	redisClient = redis.NewClient(&redis.Options{ 
		Addr: "localhost:6379", //Alamat server Redis
		Password: "",           //No password
		DB: 0,                  //Default DB
	})
}

func generateCacheKey(method, path, body string) string {
	// Hash untuk body request agar key cache unik
	hash := sha256.Sum256([]byte(body))
	bodyHash := hex.EncodeToString(hash[:])
	return fmt.Sprintf("cache:%s:%s:%s", method, path, bodyHash)
}

// Fungsi untuk menangani request cache
func cacheHandler(req server.Request) server.Response {
	//Generate cache key berdasarkan method, path, dan hash body
	cacheKey := generateCacheKey(req.Method, req.Path, req.Body)

	//Cek apakah data sdh ada di Redis
	cachedData, err := redisClient.Get(ctx, cacheKey).Result()
	ttl, _ := redisClient.TTL(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		// Cache hit
		log.Println("Cache hit, returning data from Redis.")
		return server.Response{
			StatusCode: 200,
			Body:       cachedData,
			Headers:    map[string]string{"Cache-Control": fmt.Sprintf("public,max-age=%d", ttl.Milliseconds())},
		}
	}

	// Buat request ke backend
	return server.Response{
		StatusCode: 200,
		Headers:    map[string]string{"Cache-Control": "no-cache"},
	}
}

func cacheResponseHandler(req server.Request) server.Response {
	//Generate cache key berdasarkan method, path, dan hash body
	if req.Headers["Cache-Control"] != "no-cache" {
		return server.Response{}
	}
	cacheKey := generateCacheKey(req.Method, req.Path, req.Body)

	// Simpan data ke Redis
	err := redisClient.Set(ctx, cacheKey, req.Body, 1*time.Hour).Err()
	if err != nil {
		log.Println("Error saving data to Redis:", err)
	}

	return server.Response{}
}

func main() {
	fmt.Println("Memulai plugin cache pada /tmp/cache.sock...")

	//Mulai server plugin
	go func() {
		err := server.NewServer("cache.response", cacheResponseHandler).Start() 
		if err != nil {
			fmt.Println("Error memulai server:", err)
		}
	}()
	err := server.NewServer("cache", cacheHandler).Start() 
	if err != nil {
		fmt.Println("Error memulai server:", err)
	}
}