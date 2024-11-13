package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sidra-gateway/go-pdk"
	"github.com/sidra-gateway/go-pdk/server"
)

var ctx = context.Background()
var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379", // Redis address
	Password: "    ",      // Redis password
	DB: 		0,         // Use default DB
})

// CacheHandler struct to handle cache request
type CacheHandler struct{}

// NewCacheHandler initializes the handler for server
func NewCacheHandler() *CacheHandler {
	return &CacheHandler{}
}

// Access is triggered on every request
func (h *CacheHandler) Access(pdk go.PDK) {
	url, err := pdk.Request.GetPath()
	if err != nil {
		pdk.Response.Exit(500, "Failed to retrieve URL path", nil)
		return
	}

	cacheKey := "cache:" + url
	cachedBody, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		// Return cached response if available
		headers := map[string][]string{"x-cache": {"1"}}
		pdk.Response.Exit(200, cachedBody, headers)
		return
	} else if err != redis.Nil {
		// Log and exit on Redis error
		pdk.Response.Exit(500, "Error retrieving cache: "+err.Error(), nil)
		return
	}
	// Otherwise, proceed with request; response will be cached in the header phase
}

// Header phase will cache the response after service request completes
func (h *CacheHandler) Header(pdk go.PDK) {
	statusCode, _ := pdk.Response.GetStatus()
	if statusCode == 200 {
		body, _ := pdk.Response.GetRawBody()
		url, _ := pdk.Request.GetPath()
		cacheKey := "cache:" + url

	// Store the response in Redis cache for 5 minutes
	err := rdb.Set(ctx, cacheKey, body, 5*time.Minute).Err()
	if err != nil {
		fmt.Println("Error caching response:", err)
		}
	}
}

func main() {
	server.NewServer(NewCacheHandler, "unix", "tmp/cache.sock").Start()