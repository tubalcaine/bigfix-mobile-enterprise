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

## Quick Start

1. **Build the server**:
   ```bash
   go build -o bem ./cmd/bem/
   ```

2. **Configure settings** (see `cmd/bem/bem.json`):
   ```json
   {
     "listen_port": 17967,
     "bigfix_servers": [...],
     "registration_dir": "./registrations",
     "requests_dir": "./requests"
   }
   ```

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
- `listen_port` - HTTP server port (default: 17967)
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

### TLS Settings (Optional)
- `cert_path` - Path to TLS certificate file
- `key_path` - Path to TLS private key file

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
  "debug": 0,
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
  "registration_dir": "./registrations",
  "requests_dir": "./requests"
}
```

## Interactive CLI Commands

When the BEM server is running, you can use these interactive commands:

- `cache` - Display all cached URLs for each server
- `summary` - Show cache statistics (item counts, memory usage)
- `write` - Export cache to a JSON file
- `makekey` - Generate a new RSA key pair for testing
- `registrations` - Display registration requests, clients, and active sessions
- `reload` - Re-populate cache with core types from all servers
- `help` - Display available commands
- `exit` - Terminate the server

## Debug Logging

Debug logging can be controlled via the `debug` configuration setting:

### Enable Debug Logging
Set `"debug": 1` in `bem.json` to enable detailed logging:
- Client registration and authentication events
- File monitoring operations
- Cache request processing
- XML/JSON parsing errors
- Connection acquisition and release

### Disable Debug Logging
Set `"debug": 0` in `bem.json` to disable debug output (production mode)

**Note**: Error and warning messages are always logged regardless of the debug setting.

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

Built with Go 1.19+ using:
- **Gin**: HTTP web framework
- **fsnotify**: File system monitoring
- **Crypto**: RSA key generation and validation

See individual directory README files for detailed component documentation.
