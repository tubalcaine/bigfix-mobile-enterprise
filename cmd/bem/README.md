# BEM Server Application Design

## Overview

The BEM (BigFix Enterprise Mobile) server is the main application component that provides a secure HTTPS REST API gateway to BigFix servers. It handles authentication, caching, request routing, and format conversion between mobile clients and BigFix infrastructure.

**Security Note**: The server operates in HTTPS-only mode. TLS certificate and key are required for operation.

## Files Overview

### Core Application Files

| File | Purpose | Key Components |
|------|---------|----------------|
| **main.go** | Application entry point and CLI loop | Server startup, configuration loading, interactive commands |
| **types.go** | Data structure definitions | Config structs, OTP/Client types, global state variables |
| **endpoints.go** | HTTP API route handlers | REST endpoints for registration, authentication, queries |
| **auth.go** | Authentication and session management | RSA key validation, cookie sessions, client authentication |
| **registration.go** | Client registration logic | Registration file monitoring, OTP management, key generation |
| **requests.go** | Registration request handling | Request file creation, filename sanitization |
| **storage.go** | Persistent data management | JSON file I/O for OTPs, clients, backups |
| **logging.go** | Logging configuration and setup | slog initialization, log levels, file rotation configuration |
| **server.go** | HTTP/TLS server with logging | Custom TLS server, connection logging, handshake error capture |
| **middleware.go** | Gin HTTP middleware | Request logging, error logging, panic recovery |

### Configuration & Data Files

| File | Purpose | Format |
|------|---------|--------|
| **bem.json** | Server configuration | JSON config with server settings, BigFix connections |
| **registered_clients.json** | Active client registrations | JSON array of registered clients with public keys |
| **registration_otps.json** | Pending registration OTPs | JSON array of one-time keys awaiting consumption |
| **registration_otps.json.bak.\*** | Automatic backups | Backup files created during OTP updates |

### Directory Structure

| Directory | Purpose | File Types |
|-----------|---------|------------|
| **registrations/** | Admin OTP drop folder | JSON files with completed registration requests |
| **requests/** | Client request storage | JSON files from `/requestregistration` endpoint |

## Key Components

### 1. HTTP Server (`endpoints.go`)

**Public Endpoints (no authentication):**
- `GET /requestregistration?ClientName=X` - Request device registration
- `POST /register` - Complete registration with OTP and get private key
- `GET /otp?OneTimeKey=X` - Create browser admin session
- `GET /help` - API documentation

**Protected Endpoints (require authentication):**
- `GET /urls?url=X` - Cached BigFix query responses
- `GET /servers` - List available BigFix servers
- `GET /summary` - Cache usage statistics
- `GET /cache` - Raw cache contents

### 2. Authentication System (`auth.go`)

**Client Key Authentication:**
- RSA private key validation
- Authorization header: `Authorization: Client <base64_private_key>`
- Automatic key expiration handling
- Last-used time tracking

**Browser Session Authentication:**
- Cookie-based sessions (`bem_session`)
- 8-hour session lifetime
- Session token generation and validation

### 3. Registration Management (`registration.go`, `requests.go`)

**Registration Flow:**
1. Client calls `/requestregistration` → creates request file
2. Admin processes request → moves to `registrations/` folder
3. File watcher detects new registration → loads OTPs into memory
4. Client calls `/register` with OTP → receives private key

**File Monitoring:**
- Real-time monitoring of `registrations/` directory
- Automatic processing and deletion of registration files
- Goroutine-based event handling with proper cleanup

### 4. Data Persistence (`storage.go`)

**Features:**
- Automatic backup creation (`.bak.N` files)
- Atomic file operations (write to temp, then rename)
- JSON marshaling with pretty formatting
- Error handling and recovery

**Data Types:**
- `registrationOTPs []RegistrationOTP` - Pending registrations
- `registeredClients []RegisteredClient` - Active clients
- `activeSessions map[string]time.Time` - Browser sessions

## Interactive CLI Commands

Access via the server console after starting:

### Cache Management Commands

**`cache`** - Display detailed cache information with pagination
- Shows all cached URLs for each server with comprehensive metadata
- **Displays per-item details:**
  - MaxAge: Cache lifetime in seconds
  - Content Hash: MD5 hash (truncated for readability)
  - Remaining Time: Seconds until expiration (0 if expired)
  - Hit Count: Number of times served from cache
  - Miss Count: Number of times fetched from server
- **Pagination controls:**
  - Press **ENTER** to show next page (10 items per page)
  - Type **'c' then ENTER** to continue without pausing (dump all remaining items)

**`summary`** - Show comprehensive cache statistics
- **Per-server statistics:**
  - Total item counts (total, expired, current)
  - RAM usage (KB and MB)
  - **MaxAge range:** Minimum and maximum cache lifetimes
  - **Cache hits:** Total number of cache hits across all items
  - **Cache misses:** Total number of cache misses requiring server fetches
- Useful for identifying cache performance and optimization opportunities

### Other Commands

| Command | Description | Output |
|---------|-------------|--------|
| `write` | Export cache to file | Prompts for filename, writes JSON |
| `makekey` | Generate RSA key pair | Creates `.key` and `.pub` files |
| `registrations` | Show registration status | Lists OTPs, clients, and sessions |
| `reload` | Re-populate cache | Fetches core types from all servers |
| `help` | Display available commands | Command descriptions |
| `exit` | Terminate server | Graceful shutdown |
| `<url>` | Query specific URL | Retrieve and display cached response |

## Configuration (`bem.json`)

```json
{
  "app_cache_timeout": 3600,
  "listen_port": 17967,
  "cert_path": "/path/to/cert.crt",
  "key_path": "/path/to/private.key",
  "keysize": 2048,
  "debug": 1,
  "registration_dir": "./registrations",
  "requests_dir": "./requests",
  "registration_data_dir": "./data",
  "log_to_file": true,
  "log_file_path": "./logs/bem.log",
  "log_max_size_mb": 50,
  "log_max_backups": 10,
  "log_max_age_days": 30,
  "log_compress": true,
  "log_to_console": true,
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

## Security Features

- **HTTPS-Only**: TLS required for all connections (HTTP not supported)
- **RSA Key Authentication**: 2048-bit keys with configurable lifespan
- **Request Sanitization**: Filename security, directory traversal prevention
- **Session Management**: Secure cookie handling, automatic cleanup
- **TLS 1.2+**: Minimum TLS version 1.2 with strong cipher suites
- **Input Validation**: Parameter validation, error handling

## Development Notes

- **Thread Safety**: Uses `sync.RWMutex` for concurrent access
- **Error Handling**: Comprehensive error logging and user feedback
- **Structured Logging**: Uses Go's standard `log/slog` for structured, level-based logging
- **Log Rotation**: Automatic log file rotation using lumberjack (size and age-based)
- **TLS Diagnostics**: Custom TLS server wrapper captures handshake errors with full context
- **Graceful Shutdown**: Proper resource cleanup on exit
- **File Monitoring**: Real-time registration processing with goroutine management
- **Backup Strategy**: Automatic data backups prevent data loss