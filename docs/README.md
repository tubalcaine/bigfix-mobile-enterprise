# BigFix Mobile Enterprise (BEM) - Design Documentation

## Overview

BigFix Mobile Enterprise (BEM) is a Go-based caching proxy server that provides REST API access to multiple BigFix servers. It acts as an intelligent middleware layer between mobile applications and BigFix infrastructure, offering caching, connection pooling, and unified API access.

## Architecture Summary

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Android Apps   │───▶│  BEM Server     │───▶│  BigFix Servers │
│                 │    │  (Go Proxy)     │    │  (Multiple)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Key Features

- **Multi-Server Caching**: Intelligent caching layer for multiple BigFix servers
- **Connection Pooling**: Efficient HTTP connection management
- **URL Encoding**: Proper handling of BigFix query parameters
- **Authentication**: RSA key-based client authentication with session management
- **Format Detection**: Automatic handling of JSON vs XML BigFix responses
- **CLI Interface**: Interactive command-line interface for management

## Documentation Structure

- [**Architecture Overview**](architecture.md) - System design and component interactions
- [**API Design**](api-design.md) - REST endpoints and request/response formats
- [**Deployment Guide**](deployment.md) - Installation and configuration
- [**Development Guide**](development.md) - Development environment setup

## Component Documentation

- [**BEM Server**](../cmd/bem/README.md) - Main server application design
- [**BigFix REST Client**](../pkg/bfrest/README.md) - BigFix integration library

## Quick Start

1. **Build the server**: `go build -o bem ./cmd/bem/`
2. **Configure**: Edit `bem.json` with your BigFix server details
3. **Run**: `./bem -c bem.json`
4. **Register clients**: Use the registration flow for mobile apps

## Key Technologies

- **Go 1.19+**: Core implementation language
- **Gin Framework**: HTTP server and routing
- **JSON/XML Processing**: BigFix data format handling
- **RSA Cryptography**: Client authentication
- **Sync Maps**: Thread-safe caching

## Security Features

- **RSA Key Authentication**: Each client has unique key pairs
- **Session Management**: Cookie-based admin sessions
- **Connection Validation**: URL matching and validation
- **Secure Storage**: Encrypted client credentials

## Performance Features

- **Intelligent Caching**: Per-server cache with configurable expiration
- **Connection Pooling**: Reusable HTTP connections to BigFix servers
- **Concurrent Processing**: Thread-safe operations
- **Background Operations**: Asynchronous cache population

---

For detailed information on each component, please refer to the specific documentation linked above.