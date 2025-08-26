# BigFix Mobile Enterprise - Development Guide

## Development Environment Setup

### Prerequisites

- **Go**: Version 1.19 or later
- **Git**: For version control
- **IDE**: VS Code, GoLand, or similar with Go support
- **BigFix Server**: Access to development/test BigFix environment
- **Tools**: curl, jq for API testing

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd bigfix-mobile-enterprise

# Initialize Go modules (if needed)
go mod tidy

# Verify dependencies
go mod verify
```

### Development Configuration

Create `bem-dev.json`:
```json
{
  "port": 17967,
  "keySize": 2048,
  "registrationDataDir": "dev-data",
  "servers": [
    {
      "url": "https://dev-bigfix:52311",
      "username": "admin", 
      "password": "devpassword",
      "poolsize": 5,
      "maxage": 60
    }
  ]
}
```

**Development Settings**:
- Smaller key size (2048) for faster generation
- Short cache TTL (60s) for testing
- Separate data directory
- Lower connection pool size

## Project Structure

```
bigfix-mobile-enterprise/
├── cmd/bem/                    # Main application
│   ├── main.go                # Server entry point
│   ├── endpoints.go           # HTTP route handlers
│   ├── auth.go               # Authentication logic
│   ├── registration.go       # Client registration
│   └── storage.go            # Data persistence
├── pkg/bfrest/               # BigFix REST client library
│   ├── bfcache.go           # Caching implementation
│   ├── bfconnection.go      # Connection management
│   ├── besapi.go            # API response structures
│   ├── bes.go               # BES content structures
│   └── bfrest.go            # Main library interface
├── docs/                     # Documentation
├── dev-data/                # Development data
├── registrations/           # Pending registrations
└── bem-dev.json            # Development config
```

## Building & Running

### Development Build

```bash
# Build for development (with debug info)
go build -race -o bem ./cmd/bem/

# Run with development config
./bem -c bem-dev.json

# Run with race detection
go run -race ./cmd/bem/ -c bem-dev.json
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/bfrest/

# Run tests with verbose output
go test -v ./pkg/bfrest/
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run
```

## Development Workflow

### 1. Setting Up Test Environment

**Mock BigFix Server** (optional):
```go
// test/mock_server.go
func mockBigFixServer() *httptest.Server {
    return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.Path, "/api/query") {
            w.Header().Set("Content-Type", "application/json")
            w.Write([]byte(`{"result":[["Test Action",123,"Open"]]}`))
        }
    }))
}
```

### 2. Development Server Startup

```bash
# Terminal 1: Start BEM server
./bem -c bem-dev.json

# Terminal 2: Monitor logs
tail -f bem.log

# Terminal 3: Test API endpoints
curl http://localhost:17967/help
```

### 3. Interactive Testing

**BEM CLI Commands**:
```bash
# In BEM server console
> help                    # Show available commands
> cache                   # Show cache status
> summary                 # Cache statistics
> registrations          # Show pending registrations
> servers                # List configured servers
> makekey                # Generate test key pair
```

## API Testing

### 1. Registration Flow Testing

**Step 1: Generate Registration Request**
```bash
curl "http://localhost:17967/requestregistration?ClientName=TestDevice"
```

**Step 2: Approve Registration** (move file and restart, or use CLI)

**Step 3: Complete Registration**
```bash
curl -X POST http://localhost:17967/register \
  -H "Content-Type: application/json" \
  -d '{"ClientName": "TestDevice", "OneTimeKey": "generated-key"}'
```

### 2. Authentication Testing

**Extract Private Key** from registration response and test:
```bash
# Save private key to file
echo "-----BEGIN RSA PRIVATE KEY-----..." > test-key.pem

# Base64 encode for Authorization header
ENCODED_KEY=$(base64 -w 0 test-key.pem)

# Test authenticated endpoint
curl -H "Authorization: Client $ENCODED_KEY" \
     http://localhost:17967/servers
```

### 3. BigFix Query Testing

```bash
# Test query endpoint
QUERY_URL="https://dev-bigfix:52311/api/query?output=json&relevance=names%20of%20bes%20computers"

curl -X POST http://localhost:17967/urls \
  -H "Authorization: Client $ENCODED_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"$QUERY_URL\"}"
```

## Code Development Guidelines

### 1. Go Code Style

**Package Structure**:
```go
// Package comment
package main

// Imports grouped: stdlib, third-party, local
import (
    "fmt"
    "time"
    
    "github.com/gin-gonic/gin"
    
    "github.com/tubalcaine/bigfix-mobile-enterprise/pkg/bfrest"
)
```

**Error Handling**:
```go
// Always handle errors explicitly
result, err := someFunction()
if err != nil {
    log.Printf("Error in someFunction: %v", err)
    return nil, fmt.Errorf("operation failed: %w", err)
}
```

**Logging Standards**:
```go
// Use structured logging
log.Printf("Cache hit for URL: %s (size: %d bytes)", url, size)

// Debug information in development
fmt.Printf("DEBUG: Processing request for client: %s\n", clientName)
```

### 2. Concurrency Patterns

**Thread-Safe Operations**:
```go
// Use sync.Map for concurrent access
type Cache struct {
    data *sync.Map
    mu   sync.RWMutex
}

// Proper mutex usage
func (c *Cache) Update(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data.Store(key, value)
}
```

**Connection Pool Management**:
```go
// Always release connections
conn, err := pool.Acquire()
if err != nil {
    return err
}
defer pool.Release(conn) // Ensure cleanup

// Use connections
result, err := conn.Get(url)
```

### 3. Testing Patterns

**Unit Test Structure**:
```go
func TestCacheGet(t *testing.T) {
    // Setup
    cache := NewCache()
    testURL := "https://test:52311/api/computers"
    
    // Exercise
    result, err := cache.Get(testURL)
    
    // Verify
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Contains(t, result.Json, "computer")
}
```

**Integration Test Example**:
```go
func TestEndpointIntegration(t *testing.T) {
    // Setup test server
    router := gin.New()
    setupRoutes(router, testCache, testConfig)
    
    // Test request
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/servers", nil)
    req.Header.Set("Authorization", "Client "+testKey)
    
    // Execute
    router.ServeHTTP(w, req)
    
    // Verify
    assert.Equal(t, 200, w.Code)
}
```

## Debugging

### 1. Common Development Issues

**Import Cycles**:
```bash
# Check for import cycles
go list -json ./... | jq -r '.ImportPath + " imports " + (.Imports[]? // empty)'
```

**Race Conditions**:
```bash
# Run with race detector
go run -race ./cmd/bem/ -c bem-dev.json
```

**Memory Leaks**:
```bash
# Profile memory usage
go tool pprof http://localhost:17967/debug/pprof/heap
```

### 2. Debug Configuration

**Enable Debug Mode**:
```bash
# Set Gin to debug mode
export GIN_MODE=debug

# Run with verbose logging
./bem -c bem-dev.json -v
```

**Add Debug Endpoints** (development only):
```go
// In setupRoutes() for development builds
if gin.Mode() == gin.DebugMode {
    r.GET("/debug/cache", handleDebugCache)
    r.GET("/debug/connections", handleDebugConnections)
}
```

### 3. Performance Profiling

**Enable pprof**:
```go
import _ "net/http/pprof"

// Add pprof endpoints in debug mode
if gin.Mode() == gin.DebugMode {
    pprof.Register(r)
}
```

**Profile Usage**:
```bash
# CPU profile
go tool pprof http://localhost:17967/debug/pprof/profile

# Memory profile  
go tool pprof http://localhost:17967/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:17967/debug/pprof/goroutine
```

## Contributing

### 1. Code Review Checklist

- [ ] Code follows Go formatting standards (`go fmt`)
- [ ] All errors are properly handled
- [ ] Tests added for new functionality
- [ ] Documentation updated
- [ ] Race conditions avoided
- [ ] Memory leaks prevented
- [ ] Logging appropriate for debug/production

### 2. Git Workflow

```bash
# Create feature branch
git checkout -b feature/new-functionality

# Make changes and commit
git add .
git commit -m "Add new functionality: description"

# Push and create PR
git push origin feature/new-functionality
```

### 3. Testing Before Commit

```bash
# Full test suite
go test ./...

# Race detection
go test -race ./...

# Build verification
go build ./cmd/bem/

# Format check
go fmt ./...
```

## IDE Configuration

### VS Code Settings

**.vscode/settings.json**:
```json
{
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.testFlags": ["-v"],
    "go.buildTags": "development",
    "files.eol": "\n"
}
```

### Recommended Extensions

- **Go** (official Go extension)
- **REST Client** (for API testing)
- **GitLens** (Git history/blame)
- **Thunder Client** (API testing alternative)

## Useful Development Scripts

### build.sh
```bash
#!/bin/bash
echo "Building BEM server..."
go build -race -o bem ./cmd/bem/
echo "Build complete: ./bem"
```

### test.sh
```bash
#!/bin/bash
echo "Running tests..."
go test -race -cover ./...
echo "Running vet..."
go vet ./...
echo "Tests complete"
```

### dev-setup.sh
```bash
#!/bin/bash
echo "Setting up development environment..."
mkdir -p dev-data/registrations
cp bem-dev.json.example bem-dev.json
echo "Edit bem-dev.json with your BigFix server details"
echo "Run: ./bem -c bem-dev.json"
```

This development guide provides comprehensive information for setting up a development environment, understanding the codebase, and contributing effectively to the project.