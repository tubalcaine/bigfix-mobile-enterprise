# BigFix REST Client Library - Design Documentation

## Overview

A comprehensive Go library for interacting with BigFix REST APIs, featuring intelligent caching, connection pooling, and multi-server support. This package provides the core functionality for the BEM server's BigFix connectivity, handling format detection, URL encoding, and response processing.

## Files Overview

| File | Purpose | Key Components |
|------|---------|----------------|
| **bfrest.go** | Main package interface and cache management | Public API, cache initialization, server management |
| **bfcache.go** | Caching implementation and cache operations | Cache storage, expiration logic, thread-safe operations |
| **bfconnection.go** | HTTP connection handling and pooling | Connection pools, HTTP clients, request execution |
| **besapi.go** | BigFix API specific functionality | Query execution, response parsing, error handling |
| **bes.go** | BigFix data structures and XML processing | Data models, XML unmarshaling, type definitions |
| **test/test_bfcache.go** | Unit tests for cache functionality | Cache testing, validation, performance tests |

## Core Architecture

### Cache Management (`bfcache.go`)

**BigFixCache Structure:**
```go
type BigFixCache struct {
    ServerCache   sync.Map              // Server name → BigFixServerCache
    AppCacheTO    uint64               // Global cache timeout
}

type BigFixServerCache struct {
    ServerName    string               // Server identifier
    CacheMap      sync.Map            // URL → CacheItem
    MaxAge        uint64              // Per-server cache timeout
    ConnectionPool *ConnectionPool     // HTTP connection pool
}

type CacheItem struct {
    Json         string              // Cached JSON response
    RawXML       string             // Original XML response
    Timestamp    int64              // Unix timestamp of cache entry creation
    MaxAge       uint64             // Current maximum age in seconds
    BaseMaxAge   uint64             // Base maximum age from server config
    ContentHash  string             // MD5 hash of raw server response
    HitCount     uint64             // Number of times served from cache
    MissCount    uint64             // Number of times fetched from server
}
```

**Key Features:**
- **Thread-Safe Operations**: Uses `sync.Map` for concurrent access
- **Per-Server Caching**: Individual cache settings per BigFix server
- **Automatic Expiration**: Age-based cache invalidation
- **Dual Format Storage**: Both JSON and XML response caching
- **Memory Efficient**: Lazy loading and cleanup

### Connection Management (`bfconnection.go`)

**Connection Pooling:**
```go
type ConnectionPool struct {
    client    *http.Client          // Reusable HTTP client
    poolSize  int                   // Maximum concurrent connections
    semaphore chan struct{}         // Connection limiting
}
```

**Features:**
- **Connection Reuse**: HTTP/1.1 keep-alive support
- **Concurrent Limiting**: Configurable connection pools per server
- **Timeout Management**: Request and response timeouts
- **Error Handling**: Retry logic and failure recovery

### BigFix API Integration (`besapi.go`)

**Query Execution:**
- **Relevance Query Processing**: Execute BigFix relevance expressions
- **Response Transformation**: XML to JSON conversion
- **Error Handling**: BigFix-specific error parsing
- **Authentication**: Basic auth and session management

**Supported Operations:**
- GET requests to `/api/query` endpoints
- Custom relevance query execution
- Response caching and retrieval
- Server health checking

### Data Models (`bes.go`)

**XML Schema Support:**
- **Auto-generated Structures**: From BigFix XSD schemas
- **XML Unmarshaling**: Automatic XML parsing
- **Type Safety**: Strongly typed Go structs
- **Flexible Parsing**: Support for various BigFix response formats

## Public API

### Cache Operations

```go
// Initialize cache with global timeout
cache := bfrest.GetCache(3600) // 1 hour timeout

// Add BigFix server
cache.AddServer(
    "https://bigfix-server:52311",
    "username",
    "password", 
    5 // pool size
)

// Execute cached query
item, err := cache.Get("https://bigfix-server:52311/api/query?relevance=...")

// Populate common queries
go cache.PopulateCoreTypes("https://bigfix-server:52311", 1800)
```

### Cache Management

```go
// Server iteration
cache.ServerCache.Range(func(key, value interface{}) bool {
    server := value.(*bfrest.BigFixServerCache)
    // Process server
    return true
})

// Cache item access
server.CacheMap.Range(func(key, value interface{}) bool {
    url := key.(string)
    item := value.(*bfrest.CacheItem)
    // Process cached item
    return true
})
```

## Configuration Integration

The library integrates with BEM server configuration:

```json
{
  "app_cache_timeout": 3600,
  "bigfix_servers": [
    {
      "url": "https://bigfix-server:52311",
      "username": "console_user", 
      "password": "password",
      "maxage": 1800,
      "poolsize": 5
    }
  ]
}
```

**Configuration Mapping:**
- `app_cache_timeout` → Global cache timeout
- `url` → Server endpoint
- `username/password` → Authentication credentials  
- `maxage` → Per-server cache timeout (seconds)
- `poolsize` → HTTP connection pool size

## Cache Behavior

### Expiration Logic

```go
// Cache hit check
if time.Now().Unix()-item.Timestamp > int64(server.MaxAge) {
    // Cache expired - fetch fresh data
} else {
    // Cache valid - return cached data
}
```

### Dynamic Cache Extension

When a cache item expires, the library fetches fresh data and compares the content:

- **Content Unchanged** (matching MD5 hash):
  - Cache lifetime is extended (MaxAge increases)
  - Maximum lifetime enforced by `max_cache_lifetime` config setting
  - Timestamp is updated to current time
  - HitCount increments on each cached access

- **Content Changed** (different hash):
  - Cache is replaced with new data
  - MaxAge resets to `BaseMaxAge` value
  - Timestamp resets to current time
  - MissCount increments on server fetch

This intelligent extension minimizes unnecessary BigFix server requests for stable data while ensuring fresh content is always served when changes occur.

### Cache Hit/Miss Tracking

The cache maintains comprehensive performance metrics:

- **HitCount**: Incremented each time valid cached data is returned
- **MissCount**: Incremented when data needs to be fetched from the server
- Counters persist across cache updates and garbage collection
- Useful for analyzing access patterns and optimizing `maxage` settings
- Available via CLI `summary` and `cache` commands for performance analysis

### Garbage Collection

The BEM server runs periodic garbage collection based on `garbage_collector_interval`:

- Clears JSON data from expired cache items to free memory
- Preserves metadata (including hit/miss counts) for future cache hits
- Expired items can still be extended if content hasn't changed
- No data loss - expired items are refreshed on next access

### Population Strategy

**Core Types Auto-Population:**
- Common BigFix queries are pre-cached
- Background goroutines populate cache on startup
- Reduces latency for frequent queries

**On-Demand Caching:**
- First request fetches and caches
- Subsequent requests served from cache
- Automatic background refresh with intelligent extension

## Performance Features

### Memory Management

- **Lazy Cleanup**: Expired items removed during access
- **Size Monitoring**: Cache size tracking and reporting
- **Efficient Storage**: Minimal memory overhead per cache item
- **Garbage Collection**: Periodic cleanup of expired JSON data while preserving metadata

### Network Optimization

- **Connection Pooling**: Reuse HTTP connections
- **Compression**: Automatic gzip support
- **Batching**: Multiple queries per connection when possible
- **Parallel Processing**: Concurrent server requests
- **Intelligent Caching**: Dynamic cache extension for unchanged content reduces server load

### Performance Monitoring

- **Hit/Miss Tracking**: Per-item statistics for cache effectiveness analysis
- **MaxAge Range Analysis**: Identify shortest and longest cache lifetimes
- **RAM Usage Reporting**: Track memory consumption per server
- **CLI Diagnostics**: Real-time performance insights via `summary` and `cache` commands

## Error Handling

### Network Errors
- Connection timeouts and retries
- Server unavailability handling
- Authentication failure recovery

### BigFix Errors  
- Malformed relevance query handling
- Permission denied responses
- Server overload management

### Cache Errors
- Corruption detection and recovery
- Memory pressure handling
- Thread safety guarantees

## Testing (`test/test_bfcache.go`)

**Test Coverage:**
- Cache hit/miss scenarios
- Expiration logic validation
- Concurrent access testing
- Memory leak detection
- Performance benchmarking

**Usage:**
```bash
cd pkg/bfrest
go test ./test/
```

## Integration Example

```go
package main

import "github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"

func main() {
    // Initialize cache
    cache := bfrest.GetCache(3600)
    
    // Add servers
    cache.AddServer("https://server1:52311", "user", "pass", 5)
    cache.AddServer("https://server2:52311", "user", "pass", 3)
    
    // Populate common queries
    go cache.PopulateCoreTypes("https://server1:52311", 1800)
    
    // Execute query
    url := "https://server1:52311/api/query?relevance=names of bes computers"
    result, err := cache.Get(url)
    if err != nil {
        log.Printf("Query failed: %v", err)
        return
    }
    
    fmt.Printf("Result: %s\n", result.Json)
}
```

This library provides the foundation for efficient, scalable BigFix REST API interactions with built-in caching and connection management.