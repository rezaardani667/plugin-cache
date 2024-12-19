# Plugin Cache by URL

## Description

The Cache by URL plugin enhances the performance of Sidra Api by storing backend responses in Redis based on URLs. By leveraging caching, this plugin reduces backend load and speeds up responses for repeated requests.

---

## How It Works

### Access Phase

- The plugin checks Redis for cached data using a unique key generated from:
   - **HTTP Method** (e.g., GET, POST)
   - **URL Path** (e.g., `/api/v1/resource`)
   - **Request Body Hash** to differentiate content
- If data is found (*cache hit*):
   - The plugin returns the data from Redis without forwarding the request to the backend
- If data is not found (*cache miss*):
   - The plugin forwards the request to the backend for processing

### Header Phase

- After receiving the backend response, the plugin stores it in Redis with:
   - A **unique key** as described above
   - A **TTL (Time to Live)** defaulting to 5 minutes (configurable via environment variables)

---

## Configuration

### Environment Variables

- **REDIS_ADDR**: Redis address (default: `localhost:6379`)
- **REDIS_PASSWORD**: Redis password (default: none)
- **REDIS_DB**: Redis database (default: `0`)
- **CACHE_TTL**: Cache lifetime in seconds (default: `300` seconds)

---

## How to Run

1. **Ensure Redis is running** at the specified address, e.g., `localhost:6379`
2. **Deploy the plugin** to Sidra Api's directory (e.g., `plugins/cache`)
3. Start Sidra Api with the plugin registered
4. The plugin will automatically connect using a UNIX socket (`/tmp/cache.sock`)

---

## Testing

### Endpoint

- **URL**: `http://localhost:3080/api/v1/resource`

### Testing Steps

1. Send a request to the endpoint using Postman:
    ```http
    GET http://localhost:3080/api/v1/resource
    Content-Type: application/json
    Body: {"data": "example"}
    ```
2. Observe the response and note the time taken
3. Send the same request again and observe the reduced response time

### Cache Verification

#### Cache Miss

- On the first request, data will be fetched from the backend

#### Cache Hit

- Send the same request again. Data should be returned directly from Redis

#### Verify in Redis

- Use the following commands:
   ```sh
   redis-cli KEYS cache:*
   redis-cli GET <cache_key>
   ```

### Key Notes

- **Cache TTL**: By default, the cache is stored for 5 minutes
- **Use Case**: Best suited for backend responses that rarely change, such as static or semi-static data
- **Error Handling**: If Redis fails to store data, the plugin still forwards the backend response

---

## License

MIT License

This documentation provides a comprehensive explanation of how the Cache by URL plugin works, how to configure it, how to test it with Postman, and how to verify cache in Redis. It serves as a clear guide for implementing and maintaining the cache plugin in Sidra API.
