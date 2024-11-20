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
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalln("Error connecting to Redis:", err)
	}
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

func cacheResponseHandler(req server.Request) server.Response {
	//Generate cache key berdasarkan method, path, dan hash body
		cacheKey := generateCacheKey(req.Method, req.Path, req.Body)
		err := redisClient.Set(ctx, cacheKey, req.Body, 5*time.Minute).Err()
		if err != nil {
			log.Println("Error saving data to Redis:", err)
	} else {
		log.Println("Data saved to Redis with key:", cacheKey)
	}
	return server.Response{}
}

func main() {
	fmt.Println("Memulai plugin cache pada /tmp/cache.sock...")

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