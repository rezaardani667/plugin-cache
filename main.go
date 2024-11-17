package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sidra-gateway/go-pdk/server"
)

var ctx = context.Background()

// Konfigurasi Redis client
var rdb = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379", // Alamat Redis
	//Password: "",                // Password Redis
	//DB:       0,                 // Gunakan DB default
})

// Fungsi untuk menangani request cache
func cacheHandler(req server.Request) server.Response {
	//url := req.Path
	cacheKey := "cache:" + req.Path
	resp := server.Response{}

	//Cek data di Redis
	cachedBody, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		//Data ditemukan di cache
		fmt.Println("Cache ditemukan untuk URL:", cacheKey)
		resp.StatusCode = 200
		resp.Body = cachedBody
		resp.Headers = map[string]string{"x-cache": "1"}
		return resp
	} else if err != redis.Nil {
		// Jika terjadi kesalahan saat mengambil cache selain data tidak ditemukan
		//fmt.Println("Error saat mengambil cache:", err)
		resp.StatusCode = 500
		resp.Body = "Redis error: " + err.Error()
		return resp
	}

	// Jika cache tidak ditemukan, lanjutkan request ke service backend
	//fmt.Println("Cache tidak ditemukan untuk URL:", url)
	resp.StatusCode = 0 // Tanda untuk melanjutkan ke backend
	return resp
}

// Simpan respons ke Redis
func cacheResponse(cacheKey, body string) {
	err := rdb.Set(ctx, cacheKey, body, 5*time.Minute).Err()
	if err != nil {
		fmt.Println("Error menyimpan ke Redis:", err)
	}
}

func main() {
	fmt.Println("Memulai plugin cache pada /tmp/cache.sock...")

	err := server.NewServer("cache", func(req server.Request) server.Response {
		cacheKey := "cache:" + req.Path

		// Cek cache terlebih dahulu
		resp := cacheHandler(req)
		if resp.StatusCode != 0 {
			return resp // Kembalikan cache jika ada
		}

		// Lanjutkan request ke backend
		client := &http.Client{}
		backendURL := "http://localhost:7070" + req.Path
		backendReq, err := http.NewRequest(req.Method, backendURL, nil)
		if err != nil {
			return server.Response{StatusCode: 500, Body: "Backend request error: " + err.Error()}
		}

		backendResp, err := client.Do(backendReq)
		if err != nil {
			return server.Response{StatusCode: 500, Body: "Backend error: " + err.Error()}
		}
		defer backendResp.Body.Close()

		body, _ := io.ReadAll(backendResp.Body)
		cacheResponse(cacheKey, string(body)) // Simpan ke cache

		return server.Response{
			StatusCode: backendResp.StatusCode,
			Body:       string(body),
			Headers:    map[string]string{"x-cache": "0"},
		}
	}).Start()

	if err != nil {
		fmt.Println("Error memulai server:", err)
	}
}