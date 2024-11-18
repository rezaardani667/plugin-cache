package main

import (
	//"context"
	//"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

// Fungsi untuk menangani request cache
func cacheHandler(req server.Request) server.Response {
	// Ambil URL request atau parameter unik sebagai key cache
	cacheKey := fmt.Sprintf("cache:%s:%s", req.Method, req.Path)

	//Cek apakah data sdh ada di Redis
	cachedData, err := redisClient.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		//Data tdk ditemukan di Redis, lanjut ke backend
		log.Println("cache miss, forwarding request to backend")
		client := &http.Client{}
		backendURL := "http://localhost:7070" + req.Path
		backendReq, err := http.NewRequest(req.Method, backendURL, nil)
		if err != nil {
			return server.Response{StatusCode: 500, Body: "Backend request error: " + err.Error()}
		}

		//Kirim request ke backend
		backendResp, err := client.Do(backendReq)
		if err != nil {
			return server.Response{StatusCode: 500, Body: "Backend error: " + err.Error()}
		}
		defer backendResp.Body.Close()

		// Baca respons dari backend
		body, err := io.ReadAll(backendResp.Body)
		if err != nil {
			return server.Response{StatusCode: 500, Body: "Error reading backend response: " + err.Error()}
		}

		// Simpan ke Redis dengan TTL 5 menit
		err = redisClient.Set(ctx, cacheKey, string(body), 5*time.Minute).Err()
		if err != nil {
			log.Printf("Failed to set cache: %v", err)
		}

		// Kembalikan respons dari backend
		return server.Response{
			StatusCode: backendResp.StatusCode,
			Body:       string(body),
			Headers:    map[string]string{"x-cache": "0"}, // Menandakan respons dari backend
		}
	} else if err != nil {
		// Redis error
		return server.Response{StatusCode: 500, Body: "Redis error: " + err.Error()}
	} else {
		// Cache hit
		log.Println("Cache hit, returning data from Redis.")
		return server.Response{
			StatusCode: 200,
			Body:       cachedData,
			Headers:    map[string]string{"x-cache": "1"}, // Menandakan respons dari cache
		}
	}
}

func main() {
	fmt.Println("Memulai plugin cache pada /tmp/cache.sock...")

	//Mulai server plugin
	err := server.NewServer("cache", cacheHandler).Start() 
	//{
	// 	cacheKey := fmt.Sprintf("cache:%s:%s", req.Method, req.Path)

	// 	// Cek cache terlebih dahulu
	// 	resp := cacheHandler(req)
	// 	if resp.StatusCode != 0 {
	// 		return resp // Kembalikan cache jika ada
	// 	}

	// 	// Lanjutkan request ke backend jk cache tidak ditemukan
	// 	client := &http.Client{}
	// 	backendURL := "http://localhost:7070" + req.Path
	// 	backendReq, err := http.NewRequest(req.Method, backendURL, nil)
	// 	if err != nil {
	// 		return server.Response{StatusCode: 500, Body: "Backend request error: " + err.Error()}
	// 	}

	// 	backendResp, err := client.Do(backendReq)
	// 	if err != nil {
	// 		return server.Response{StatusCode: 500, Body: "Backend error: " + err.Error()}
	// 	}
	// 	defer backendResp.Body.Close()

	// 	body, _ := io.ReadAll(backendResp.Body)
	// 	cacheResponse(cacheKey, string(body)) // Simpan respons backend ke Redis

	// 	return server.Response{
	// 		StatusCode: backendResp.StatusCode,
	// 		Body:       string(body),
	// 		Headers:    map[string]string{"x-cache": "0"}, // Header untuk menandai respons langsung dari backend
	// 	}
	// }).Start()

	if err != nil {
		fmt.Println("Error memulai server:", err)
	}
}