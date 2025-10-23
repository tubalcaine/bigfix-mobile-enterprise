# BigFix Enterprise Mobile (BEM) Server

A Go-based caching proxy server for BigFix REST APIs, designed to support mobile applications with multi-server connectivity, client authentication, and intelligent caching.

## Project Structure

```
bigfix-mobile-enterprise/
├── cmd/bem/                 # Main application executable and source code
├── pkg/bfrest/              # BigFix REST API client library
├── training_xml/            # Training data and XML generation scripts
└── xmlschema/              # XML Schema definitions for BigFix APIs
```

## Key Features

- **Multi-Server Support**: Connect to multiple BigFix servers simultaneously
- **Intelligent Caching**: Configurable cache timeouts per server for performance optimization
- **Client Authentication**: RSA key-based authentication for mobile clients
- **Registration System**: Self-service device registration with admin approval
- **REST API Proxy**: Caches BigFix `/api/query` responses
- **Interactive CLI**: Real-time management and diagnostics
- **HTTPS-Only**: Secure TLS-only communication (HTTP not supported)

## Quick Start

1. **Build the server**:
   ```bash
   go build -o bem ./cmd/bem/
   ```

2. **Configure settings** (see `cmd/bem/bem.json`):
   ```json
   {
     "listen_port": 17967,
     "cert_path": "./bem-cert.pem",
     "key_path": "./bem-key.pem",
     "bigfix_servers": [...],
     "registration_dir": "./registrations",
     "requests_dir": "./requests"
   }
   ```

   **Note**: TLS certificate and key are required. The server will not start without them.

3. **Run the server**:
   ```bash
   ./bem -c bem.json
   ```

## Architecture

The BEM server acts as a caching proxy between mobile applications and BigFix servers:

```
Mobile App → BEM Server → BigFix Server(s)
    ↓           ↓              ↓
   Auth      Cache         REST API
```

## Authentication Flow

1. Mobile app requests registration (`/requestregistration`)
2. Admin processes request and provides one-time key
3. Mobile app completes registration (`/register`) and receives private key
4. Subsequent API calls use private key authentication

## API Endpoints

- `/requestregistration` - Request device registration
- `/register` - Complete device registration with OTP
- `/otp` - Browser-based admin session creation
- `/urls` - Cached BigFix query responses with metadata (authenticated)
- `/servers` - List available BigFix servers (authenticated)
- `/summary` - Cache statistics (authenticated)
- `/cache` - List cached URLs per server (authenticated)
- `/help` - API documentation

### `/urls` Endpoint Response Format

The `/urls` endpoint returns comprehensive cache metadata along with the requested data:

```json
{
  "cacheitem": "...",           // Actual cached data (JSON or XML-to-JSON)
  "iscachehit": true,           // Whether data was served from valid cache
  "timestamp": 1729169340,      // Unix epoch timestamp of cache entry creation
  "maxage": 300,                // Current maximum age in seconds
  "ttl": 245,                   // Time-to-live in seconds until expiration
  "hitcount": 5,                // Number of times this URL was served from cache
  "misscount": 2,               // Number of times this URL required server fetch
  "contenthash": "a1b2c3..."    // MD5 hash of the raw server response
}
```

This metadata enables clients to make informed decisions about cache freshness and reliability.

## Configuration

The server uses JSON configuration files (`bem.json`) with the following options:

### Core Settings
- `listen_port` - HTTPS server port (default: 17967)
- `keysize` - RSA key size for client registration (default: 2048)
- `debug` - Debug logging control (0 = off, non-zero = on)

### Cache Settings
- `app_cache_timeout` - Global cache timeout in seconds (0 = use per-server settings)
- `max_cache_lifetime` - Maximum cache lifetime in seconds (default: 86400 = 24 hours)
- `garbage_collector_interval` - Cache cleanup interval in seconds (default: 15)

### Directory Settings
- `registration_dir` - Directory for monitoring registration OTP files (e.g., "./registrations")
- `requests_dir` - Directory for client registration requests (e.g., "./requests")
- `registration_data_dir` - Directory for persistent registration data storage

### TLS Settings (Required)
- `cert_path` - Path to TLS certificate file (required)
- `key_path` - Path to TLS private key file (required)

**Note**: The BEM server operates in HTTPS-only mode. TLS certificate and key must be provided. HTTP connections are not supported.

### Logging Configuration
- `log_to_file` - Enable file logging (default: false)
- `log_file_path` - Path to log file (default: "./logs/bem.log")
- `log_max_size_mb` - Maximum log file size in MB before rotation (default: 100)
- `log_max_backups` - Maximum number of old log files to retain (default: 5)
- `log_max_age_days` - Maximum number of days to retain old log files (default: 30)
- `log_compress` - Compress rotated log files with gzip (default: false)
- `log_to_console` - Also log to console/stdout when file logging is enabled (default: false)

### BigFix Server Configuration
Each server in the `bigfix_servers` array supports:
- `url` - BigFix server URL (e.g., "https://server:52311")
- `username` - BigFix API username
- `password` - BigFix API password
- `maxage` - Cache expiration time for this server (seconds)
- `poolsize` - Connection pool size for this server

### Example Configuration
```json
{
  "app_cache_timeout": 0,
  "keysize": 2048,
  "debug": 1,
  "max_cache_lifetime": 86400,
  "garbage_collector_interval": 15,
  "bigfix_servers": [
    {
      "url": "https://bigfix-server-1:52311",
      "username": "admin",
      "password": "password",
      "maxage": 300,
      "poolsize": 4
    }
  ],
  "listen_port": 17967,
  "cert_path": "./bem-cert.pem",
  "key_path": "./bem-key.pem",
  "registration_dir": "./registrations",
  "requests_dir": "./requests",
  "log_to_file": true,
  "log_file_path": "./logs/bem.log",
  "log_max_size_mb": 50,
  "log_max_backups": 10,
  "log_max_age_days": 30,
  "log_compress": true,
  "log_to_console": true
}
```

## Interactive CLI Commands

When the BEM server is running, you can use these interactive commands:

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
- Example output:
  ```
  === Server: https://bigfix-server:52311 ===

    URL: https://bigfix-server:52311/api/sites
      MaxAge: 600 seconds
      Content Hash: a1b2c3d4e5f6...
      Remaining Time: 245 seconds
      Hit Count: 42
      Miss Count: 3

  --- Showing 10 of 50 items. Press ENTER for more, or 'c' then ENTER to continue without pausing:
  ```

**`summary`** - Show comprehensive cache statistics
- **Per-server statistics:**
  - Total item counts (total, expired, current)
  - RAM usage (KB and MB)
  - **MaxAge range:** Minimum and maximum cache lifetimes
  - **Cache hits:** Total number of cache hits across all items
  - **Cache misses:** Total number of cache misses requiring server fetches
- Useful for identifying cache performance and optimization opportunities
- Example output:
  ```
  For server https://bigfix-server:52311
      We have:
          150 total items, 12 expired, 138 current
          RAM usage: 2048.50 KB (2.00 MB)
          MaxAge range: 300 to 7200 seconds
          Cache hits: 1250, Cache misses: 175
  ```

### Other Commands

- `write` - Export cache to a JSON file
- `makekey` - Generate a new RSA key pair for testing
- `registrations` - Display registration requests, clients, and active sessions
- `reload` - Re-populate cache with core types from all servers
- `help` - Display available commands
- `exit` - Terminate the server

## Logging

The BEM server uses Go's standard `log/slog` library for structured logging with automatic log rotation support.

### Log Levels

Logging behavior is controlled via the `debug` configuration setting:

**Debug Mode (`"debug": 1`):**
- DEBUG level messages (detailed diagnostics)
- INFO level messages (operational information)
- WARN level messages (warnings and non-critical issues)
- ERROR level messages (errors and failures)
- Source code locations included in logs
- TLS handshake details
- HTTP request/response details
- Cache operation details

**Production Mode (`"debug": 0`):**
- INFO level messages and above
- WARN level messages
- ERROR level messages
- No source code locations
- Reduced verbosity for performance

### Structured Logging

All log messages use structured key-value pairs for easy parsing and analysis:

```
time=2025-10-20T09:37:15.005-04:00 level=ERROR msg="TLS connection error" error="tls: first record does not look like a TLS handshake" remote_addr="192.168.1.100:54321" bytes_read=5
```

### Log Rotation

When file logging is enabled (`"log_to_file": true`), logs automatically rotate based on:

- **Size-Based Rotation**: When log file reaches `log_max_size_mb`, it's rotated
- **Age-Based Cleanup**: Files older than `log_max_age_days` are deleted
- **Backup Retention**: Keep up to `log_max_backups` old log files
- **Compression**: Optionally compress old logs with gzip (`log_compress`)

**Example Rotated Logs:**
```
./logs/
├── bem.log                           (current log)
├── bem-2025-10-20T09-15-30.123.log  (rotated log)
├── bem-2025-10-19T14-22-18.456.log.gz (compressed)
└── bem-2025-10-18T08-45-02.789.log.gz (compressed)
```

### Log Destinations

**Console Only (default):**
```json
{
  "log_to_file": false
}
```
Logs appear only on stdout/console.

**File Only:**
```json
{
  "log_to_file": true,
  "log_to_console": false
}
```
Logs written only to file, silent console.

**Both File and Console:**
```json
{
  "log_to_file": true,
  "log_to_console": true
}
```
Logs appear on both console and file (recommended for development).

### TLS Connection Logging

The server includes comprehensive TLS connection logging to diagnose handshake failures:

- Connection acceptance from client IP
- TLS version and cipher suite negotiation
- Handshake completion details
- **TLS errors with full context** (client IP, error message, bytes read)

This is particularly useful for diagnosing "Unable to parse TLS header" and similar TLS connection issues.

## Cache Management

The BEM server implements intelligent caching with the following features:

### Dynamic Cache Extension
- When content is unchanged (verified by MD5 hash), cache lifetime is extended
- Maximum cache lifetime is enforced by `max_cache_lifetime` setting
- Changed content resets to the base `maxage` value

### Cache Hit/Miss Tracking
- **Hit Count**: Incremented each time valid cached data is returned
- **Miss Count**: Incremented when data needs to be fetched from the server
- Counters persist across cache updates and garbage collection
- Useful for analyzing access patterns and optimizing `maxage` settings

### Garbage Collection
- Runs periodically based on `garbage_collector_interval` setting
- Clears JSON data from expired cache items to free memory
- Preserves metadata (including hit/miss counts) for future cache hits

### Cache Inspection
Use the `/cache` endpoint or the `cache` CLI command to inspect cached URLs.
Use the `/summary` endpoint for aggregate statistics including memory usage.

## Troubleshooting

### Debug Mode
Enable debug logging (`"debug": 1`) to diagnose issues with:
- Client authentication failures
- Registration processing errors
- Cache misses or failures
- Server connection problems

### Common Issues

**Registration not working:**
- Check that `registration_dir` exists and is writable
- Verify OTP file format is correct JSON
- Enable debug logging to see file processing errors

**Cache not populating:**
- Verify BigFix server credentials in configuration
- Check network connectivity to BigFix servers
- Review cache statistics with `summary` command

**Authentication failures:**
- Verify client private key matches registered public key
- Check if client registration has expired
- Enable debug logging to see authentication attempts

## Development

Built with Go 1.21+ using:
- **Gin**: HTTP web framework
- **slog**: Structured logging (Go standard library)
- **lumberjack**: Automatic log rotation
- **fsnotify**: File system monitoring
- **Crypto**: RSA key generation and validation

See individual directory README files for detailed component documentation.
