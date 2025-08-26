# BigFix Mobile Enterprise - Architecture Design

## System Architecture

### High-Level Overview

```
┌─────────────────┐
│   Mobile Apps   │
│   (Android)     │
└─────────┬───────┘
          │ HTTPS/JSON
          ▼
┌─────────────────┐
│   BEM Server    │
│   (Go Proxy)    │
├─────────────────┤
│ • Authentication│
│ • Caching       │
│ • Connection    │
│   Pool          │
│ • URL Encoding  │
└─────────┬───────┘
          │ HTTPS/JSON+XML
          ▼
┌─────────────────┐
│  BigFix Servers │
│  (Multiple)     │
└─────────────────┘
```

## Core Components

### 1. HTTP Server Layer

**Component**: Gin HTTP Server  
**Location**: `cmd/bem/main.go`, `cmd/bem/endpoints.go`

```go
// Main server setup
router := gin.Default()
setupRoutes(router, cache, config)
server := &http.Server{
    Addr:    fmt.Sprintf(":%d", config.Port),
    Handler: router,
}
```

**Responsibilities**:
- REST API endpoint handling
- Request routing and validation
- Authentication middleware
- Response formatting

### 2. Authentication & Session Management

**Component**: Auth System  
**Location**: `cmd/bem/auth.go`, `cmd/bem/registration.go`

```go
// RSA key-based client authentication
func isValidClientKey(encodedPrivateKey string) (string, bool)

// Cookie-based admin sessions  
func createAdminSession(otp RegistrationOTP) string
```

**Features**:
- **RSA Key Pairs**: Each client has unique 2048/4096-bit keys
- **Registration Flow**: OTP-based client registration
- **Session Cookies**: 8-hour admin sessions
- **Key Validation**: Public key matching and expiration checks

### 3. Caching Layer

**Component**: BigFixCache  
**Location**: `pkg/bfrest/bfcache.go`

```go
type BigFixCache struct {
    ServerCache *sync.Map  // Thread-safe server cache
    MaxAge      uint64     // Global cache expiration
}

type BigFixServerCache struct {
    ServerName string
    CacheMap   *sync.Map  // Per-server cached items
    cpool      *Pool      // Connection pool
    MaxAge     uint64     // Server-specific expiration
}
```

**Architecture**:
- **Two-Level Cache**: Global cache containing per-server caches
- **Thread Safety**: sync.Map for concurrent access
- **TTL Management**: Per-server and per-item expiration
- **Cache Items**: Store both raw XML/JSON and processed data

### 4. Connection Pool Management

**Component**: Connection Pool  
**Location**: `pkg/bfrest/bfconnection.go`

```go
type Pool struct {
    connections chan *BFConnection  // Buffered channel pool
    factory     func() (*BFConnection, error)
    closed      bool
    mutex       sync.Mutex
}

type BFConnection struct {
    URL      string      // BigFix server URL
    Username string      // Authentication credentials
    Password string
    Conn     http.Client  // Reusable HTTP client
}
```

**Features**:
- **Buffered Channels**: Pool size configurable per server
- **Connection Reuse**: HTTP client with persistent connections
- **Timeout Handling**: 30-second acquisition timeout
- **TLS Configuration**: InsecureSkipVerify for development

### 5. Request Processing Pipeline

#### Format Detection & Handling

```go
// Smart format detection in retrieveBigFixData()
if strings.Contains(parsedURL.Path, "/api/query") {
    queryParams, err := url.ParseQuery(parsedURL.RawQuery)
    if queryParams.Get("output") == "json" || queryParams.Get("format") == "json" {
        // JSON passthrough mode
        return &CacheItem{
            RawXML: rawResponse,  // Actually JSON
            Json:   rawResponse,  // Direct passthrough
        }
    }
}
// XML parsing and conversion mode
```

#### URL Encoding Pipeline

```go
// Proper URL parameter encoding in BFConnection.Get()
encodedURL := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
if parsedURL.RawQuery != "" {
    values, err := url.ParseQuery(parsedURL.RawQuery)
    encodedURL += "?" + values.Encode()  // Proper escaping
}
```

## Data Flow Architecture

### 1. Request Flow

```
Client Request → Authentication → Cache Check → Connection Pool → BigFix Server
     ↓              ↓               ↓              ↓              ↓
Android App → RSA Key Check → Cache Hit? → Acquire Conn → HTTP Request
                ↓               ↓              ↓              ↓
            Session Valid → Return Cached → Pool Management → Response
```

### 2. Response Processing

```
BigFix Response → Format Detection → Processing → Caching → Client Response
       ↓              ↓                  ↓          ↓           ↓
   XML/JSON → output=json? → Parse/Convert → Store → Format Response
       ↓              ↓                  ↓          ↓           ↓
   Raw Data → JSON Passthrough → XML→JSON → Cache Item → JSON Response
                 ↓                  ↓          ↓           ↓
             Direct Return → Struct Parsing → TTL Set → Send to Client
```

## Concurrency Architecture

### Thread Safety

```go
// Global cache singleton with mutex protection
var cacheInstance *BigFixCache
var cacheMu = &sync.Mutex{}

// Per-server thread-safe operations
type BigFixServerCache struct {
    CacheMap *sync.Map  // Concurrent read/write safe
}

// Registration data protection
var registrationMutex sync.RWMutex
```

### Connection Pool Concurrency

```go
// Thread-safe connection acquisition
func (p *Pool) Acquire() (*BFConnection, error) {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    
    select {
    case conn := <-p.connections:
        return conn, nil
    case <-time.After(30 * time.Second):
        return nil, fmt.Errorf("timeout")
    }
}
```

## Configuration Architecture

### JSON Configuration

```json
{
  "port": 17967,
  "keySize": 4096,
  "servers": [
    {
      "url": "https://server1:52311",
      "username": "admin",
      "password": "password",
      "poolsize": 10,
      "maxage": 300
    }
  ]
}
```

### Runtime Storage

```
bigfix-mobile-enterprise/
├── bem.json                    # Server configuration
├── data/                       # Runtime data directory
│   ├── registered_clients.json # Client registrations
│   ├── registration_otps.json  # Pending registrations
│   └── *.bak.*                # Automatic backups
```

## Security Architecture

### Authentication Flow

```
1. Client Registration:
   OTP Generation → Admin Approval → Key Pair Creation → Storage

2. Request Authentication:
   Private Key → Public Key Derivation → Registered Key Match → Access Grant

3. Admin Sessions:
   OTP Validation → Session Token → HTTP Cookie → 8-hour Expiry
```

### Key Management

```go
// RSA key generation (2048/4096 bit)
privateKey, _ := rsa.GenerateKey(rand.Reader, keySize)
publicKeyPEM := pem.EncodeToMemory(&pem.Block{
    Type:  "PUBLIC KEY",
    Bytes: publicKeyBytes,
})
```

## Error Handling Architecture

### Graceful Degradation

- **Connection Failures**: Pool management with retries
- **Cache Misses**: Fallback to direct BigFix queries
- **Authentication Errors**: Clear error messages for re-registration
- **Timeout Handling**: Configurable timeouts at multiple layers

### Logging & Debugging

```go
// Structured logging throughout
log.Printf("Cache hit successful for URL: %s", url)
fmt.Printf("DEBUG.BESAPI: xml.Unmarshal failed, err [%s]", err)
```

## Performance Characteristics

### Caching Benefits

- **Cache Hit Ratio**: Configurable TTL per server (default 300s)
- **Connection Reuse**: HTTP/1.1 persistent connections
- **Memory Efficiency**: Lazy loading of cache items
- **Concurrent Access**: Lock-free reads with sync.Map

### Scalability Considerations

- **Horizontal**: Multiple BEM instances possible
- **Vertical**: Connection pool size configurable per server
- **Memory**: Cache size grows with usage patterns
- **Network**: Efficient connection pooling reduces overhead

---

This architecture provides a robust, scalable, and maintainable proxy layer between mobile applications and BigFix infrastructure, with emphasis on performance, security, and operational simplicity.