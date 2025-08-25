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
- `/urls` - Cached BigFix query responses (authenticated)
- `/servers` - List available BigFix servers (authenticated)
- `/summary` - Cache statistics (authenticated)
- `/help` - API documentation

## Configuration

The server uses JSON configuration files with support for:
- Multiple BigFix server connections
- Per-server cache settings
- TLS certificate configuration
- Registration directory monitoring
- Custom key lifespans

## Development

Built with Go 1.19+ using:
- **Gin**: HTTP web framework
- **fsnotify**: File system monitoring
- **Crypto**: RSA key generation and validation

See individual directory README files for detailed component documentation.
