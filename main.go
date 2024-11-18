package main

import (
	//"context"
	//"bytes"
	"crypto/sha256"
	//"encoding/json"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"strings"

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
	if err == nil && cachedData != "" {
		// Cache hit
		log.Println("Cache hit, returning data from Redis.")
		return server.Response{
			StatusCode: 200,
			Body:       cachedData,
			Headers:    map[string]string{"X-Cache": "1"},
		}
	}

	// Cache miss, log dan teruskan request ke backend
	log.Println("Cache miss, forwarding request to backend.")

	// Ambil body dari request client
	bodyReader := strings.NewReader(req.Body)

	// Buat request ke backend
	backendURL := "http://localhost:7070" + req.Path
	backendReq, err := http.NewRequest(req.Method, backendURL, bodyReader)
	if err != nil {
		log.Println("Error creating backend request:", err)
		return server.Response{StatusCode: 500, Body: "Backend request error: " + err.Error()}
	}

	// Salin header dari request client ke backend
	for key, value := range req.Headers {
		backendReq.Header.Set(key, value)
	}

	client := &http.Client{}
	backendResp, err := client.Do(backendReq)
	if err != nil {
		log.Println("Error sending request to backend:", err)
		return server.Response{StatusCode: 500, Body: "Backend error: " + err.Error()}
	}
	defer backendResp.Body.Close()

	// Baca respons dari backend
	body, err := io.ReadAll(backendResp.Body)
	if err != nil {
		log.Println("Error reading backend response:", err)
		return server.Response{StatusCode: 500, Body: "Error reading backend response."}
	}

	// Format cache value
	cacheValue := string(body)

	// Simpan data ke Redis dgn ttl
	err = redisClient.Set(ctx, cacheKey, cacheValue, 5*time.Minute).Err()
	if err != nil {
		log.Println("Error saving data to Redis:", err)
	}

	// Kirim respons ke client
	return server.Response{
		StatusCode: backendResp.StatusCode,
		Body:       cacheValue,
		Headers:    map[string]string{"X-Cache": "0"},
	}
}

func main() {
	fmt.Println("Memulai plugin cache pada /tmp/cache.sock...")

	//Mulai server plugin
	err := server.NewServer("cache", cacheHandler).Start() 
	if err != nil {
		fmt.Println("Error memulai server:", err)
	}
}