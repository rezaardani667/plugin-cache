package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sidra-gateway/go-pdk/server"
)

var ctx = context.Background()

// Konfigurasi Redis client
var rdb = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379", // Alamat Redis
	Password: "",                // Password Redis
	DB:       0,                 // Gunakan DB default
})

// Fungsi untuk menangani request cache
func cacheHandler(req server.Request) server.Response {
	url := req.Path
	cacheKey := "cache:" + url
	resp := server.Response{}

	// Mencoba mengambil data dari cache
	cachedBody, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		// Jika data ada di cache, kembalikan data dari cache
		fmt.Println("Cache ditemukan untuk URL:", url)
		resp.StatusCode = 200
		resp.Body = cachedBody
		resp.Headers = map[string]string{"x-cache": "1"}
		return resp
	} else if err != redis.Nil {
		// Jika terjadi kesalahan saat mengambil cache selain data tidak ditemukan
		fmt.Println("Error saat mengambil cache:", err)
		resp.StatusCode = 500
		resp.Body = "Error retrieving cache: " + err.Error()
		return resp
	}

	// Jika cache tidak ditemukan, lanjutkan request ke service backend
	fmt.Println("Cache tidak ditemukan untuk URL:", url)
	resp.StatusCode = 0 // Tanda untuk melanjutkan ke backend
	return resp
}

// Fungsi untuk menyimpan respons ke cache
func cacheHeaderHandler(resp server.Response, req server.Request) {
	if resp.StatusCode == 200 {
		url := req.Path
		cacheKey := "cache:" + url

		// Simpan respons ke Redis cache dengan durasi 5 menit
		err := rdb.Set(ctx, cacheKey, resp.Body, 5*time.Minute).Err()
		if err != nil {
			fmt.Println("Error saat menyimpan response ke cache:", err)
		} else {
			fmt.Println("Response disimpan ke cache untuk URL:", url)
		}
	}
}

func main() {
	fmt.Println("Memulai server cache plugin pada /tmp/cache.sock...")

	err := server.NewServer("cache", func(req server.Request) server.Response {
		// Memanggil handler cache untuk memeriksa cache
		resp := cacheHandler(req)
		if resp.StatusCode != 0 {
			// Jika respons ditemukan di cache, kembalikan respons
			return resp
		}

		// Jika tidak ada di cache, lanjutkan ke backend
		// Setelah mendapatkan respons dari backend, simpan ke cache
		resp.StatusCode = 200 // Misalnya respons backend sukses
		resp.Body = "Data dari backend" // Ini seharusnya hasil dari backend sebenarnya
		cacheHeaderHandler(resp, req)

		return resp
	}).Start()

	if err != nil {
		fmt.Println("Error saat memulai server:", err)
		return
	}

	fmt.Println("Server cache plugin berjalan pada /tmp/cache.sock")
	select {} // Menjaga server tetap berjalan tanpa keluar
}
