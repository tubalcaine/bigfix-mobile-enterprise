# BigFix Mobile Enterprise - Deployment Guide

## Prerequisites

### System Requirements

- **Operating System**: Linux, Windows, or macOS
- **Go Version**: 1.19 or later
- **Memory**: Minimum 512MB RAM (2GB+ recommended for production)
- **Storage**: 100MB+ for application and cache data
- **Network**: HTTPS access to BigFix servers (port 52311)

### BigFix Server Requirements

- **BigFix Version**: 9.5+ (REST API support required)
- **Credentials**: Admin account with REST API access
- **Network**: Accessible from BEM server on port 52311 (HTTPS)
- **SSL Certificate**: Valid or development certificate

## Installation

### 1. Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd bigfix-mobile-enterprise

# Build the application
go build -o bem ./cmd/bem/

# Verify build
./bem --help
```

### 2. Directory Structure Setup

```bash
# Create required directories
mkdir -p data/registrations

# Set appropriate permissions
chmod 700 data/
chmod 755 registrations/
```

**Directory Layout**:
```
bigfix-mobile-enterprise/
├── bem                     # Compiled binary
├── bem.json               # Configuration file
├── data/                  # Runtime data (auto-created)
│   ├── registered_clients.json
│   ├── registration_otps.json
│   └── *.bak.*           # Automatic backups
└── registrations/         # Pending registration requests
    └── *.json            # Client registration files
```

## Configuration

### 1. Basic Configuration File

Create `bem.json`:

```json
{
  "port": 17967,
  "keySize": 4096,
  "registrationDataDir": "data",
  "servers": [
    {
      "url": "https://bigfix-server1.example.com:52311",
      "username": "admin",
      "password": "password123",
      "poolsize": 10,
      "maxage": 300
    },
    {
      "url": "https://bigfix-server2.example.com:52311", 
      "username": "admin",
      "password": "password456",
      "poolsize": 5,
      "maxage": 600
    }
  ]
}
```

### 2. Configuration Parameters

| Parameter | Description | Default | Notes |
|-----------|-------------|---------|-------|
| `port` | HTTP server port | 17967 | Must be available |
| `keySize` | RSA key size (bits) | 4096 | 2048 or 4096 |
| `registrationDataDir` | Data storage path | "data" | Relative to binary |
| `servers[].url` | BigFix server URL | - | Include port 52311 |
| `servers[].username` | BigFix admin user | - | REST API access required |
| `servers[].password` | BigFix password | - | Store securely |
| `servers[].poolsize` | Connection pool size | 10 | Adjust based on load |
| `servers[].maxage` | Cache TTL (seconds) | 300 | Server-specific caching |

### 3. Security Considerations

**Configuration File**:
```bash
# Secure the configuration file
chmod 600 bem.json
chown bem-user:bem-group bem.json
```

**Data Directory**:
```bash
# Secure runtime data
chmod 700 data/
chown -R bem-user:bem-group data/
```

## Startup

### 1. Command Line Options

```bash
# Start with configuration file
./bem -c bem.json

# Start with custom port (overrides config)
./bem -c bem.json -p 8080

# View available options
./bem --help
```

### 2. Startup Verification

**Expected Output**:
```
BEM Server starting...
Loading configuration from: bem.json
Configured servers:
  - https://bigfix-server1.example.com:52311 (pool: 10, cache: 300s)
  - https://bigfix-server2.example.com:52311 (pool: 5, cache: 600s)
Server starting on port 17967
[GIN] Listening and serving HTTP on :17967
```

**Health Check**:
```bash
curl http://localhost:17967/help
# Should return HTML help page
```

## Client Registration Process

### 1. Generate Registration Request

From Android app or via API:
```http
GET http://bem-server:17967/requestregistration?ClientName=AndroidDevice_001
```

This creates: `registrations/AndroidDevice_001_<timestamp>.json`

### 2. Admin Approval

**Manual Process**:
1. Review registration file in `registrations/` directory
2. Move approved file to `data/registrations/`
3. Restart BEM server or use CLI command

**CLI Process**:
```bash
# In BEM server interactive mode
> registrations
# Shows pending registrations
> approve AndroidDevice_001
# Approves specific client
```

### 3. Complete Registration

Client receives OTP and completes registration:
```http
POST http://bem-server:17967/register
{
  "ClientName": "AndroidDevice_001",
  "OneTimeKey": "generated-otp-key"
}
```

## Production Deployment

### 1. Service Configuration

**systemd Service** (`/etc/systemd/system/bem.service`):
```ini
[Unit]
Description=BigFix Mobile Enterprise Server
After=network.target

[Service]
Type=simple
User=bem
Group=bem
WorkingDirectory=/opt/bem
ExecStart=/opt/bem/bem -c /opt/bem/bem.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Enable and Start**:
```bash
sudo systemctl enable bem
sudo systemctl start bem
sudo systemctl status bem
```

### 2. Reverse Proxy Setup

**Nginx Configuration**:
```nginx
server {
    listen 443 ssl;
    server_name bem.example.com;
    
    ssl_certificate /path/to/ssl/cert.pem;
    ssl_certificate_key /path/to/ssl/key.pem;
    
    location / {
        proxy_pass http://localhost:17967;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 3. Firewall Configuration

```bash
# Allow BEM server port
sudo ufw allow 17967/tcp

# Allow HTTPS (if using reverse proxy)
sudo ufw allow 443/tcp

# Allow BigFix server access (outbound)
sudo ufw allow out 52311/tcp
```

### 4. SSL/TLS Configuration

**For Production**: Use valid SSL certificates
```json
{
  "tls": {
    "enabled": true,
    "certFile": "/path/to/cert.pem",
    "keyFile": "/path/to/key.pem"
  }
}
```

**For Development**: BigFix servers with self-signed certificates are supported (InsecureSkipVerify enabled)

## Monitoring & Maintenance

### 1. Log Monitoring

**Application Logs**:
```bash
# Follow logs with systemd
journalctl -u bem -f

# Log rotation setup
sudo logrotate /etc/logrotate.d/bem
```

### 2. Health Checks

**Endpoint Monitoring**:
```bash
# Basic health check
curl -f http://localhost:17967/help || echo "Service down"

# Server list check (requires auth)
curl -H "Authorization: Client <key>" http://localhost:17967/servers
```

### 3. Backup Procedures

**Configuration Backup**:
```bash
# Backup configuration and data
tar -czf bem-backup-$(date +%Y%m%d).tar.gz bem.json data/
```

**Automatic Backups**: BEM creates automatic backups of critical files:
- `registered_clients.json.bak.N`
- `registration_otps.json.bak.N`

### 4. Cache Management

**CLI Commands**:
```bash
# In BEM interactive mode
> cache              # Show cache status
> summary            # Detailed cache statistics
> clear <server>     # Clear specific server cache (if implemented)
```

## Performance Tuning

### 1. Connection Pool Sizing

```json
{
  "servers": [{
    "poolsize": 20,    // Increase for high-load scenarios
    "maxage": 180      // Reduce for more frequent updates
  }]
}
```

**Guidelines**:
- **Low Load**: 5-10 connections per server
- **Medium Load**: 10-20 connections per server  
- **High Load**: 20-50 connections per server

### 2. Memory Management

**Cache Sizing**: Monitor memory usage and adjust TTL values:
- **Frequent Updates**: Lower `maxage` (60-180 seconds)
- **Static Data**: Higher `maxage` (600-1800 seconds)
- **Development**: Very low `maxage` (30-60 seconds)

### 3. Network Optimization

```go
// Connection timeout tuning in code
client := http.Client{
    Transport: &transport,
    Timeout:   60 * time.Second,  // Reduce for faster failure detection
}
```

## Troubleshooting

### 1. Common Issues

**Connection Refused**:
```bash
# Check if BEM server is running
ps aux | grep bem
netstat -tlnp | grep 17967

# Check firewall
sudo ufw status
```

**Authentication Failures**:
```bash
# Verify client registration
cat data/registered_clients.json

# Check key expiration
# Keys expire after keyLifespanDays (default: 365 days)
```

**BigFix Server Connectivity**:
```bash
# Test direct connection
curl -k https://bigfix-server:52311/api/help

# Check DNS resolution
nslookup bigfix-server.example.com
```

### 2. Debug Mode

Enable verbose logging:
```bash
# Run with debug output
GIN_MODE=debug ./bem -c bem.json
```

### 3. Performance Issues

**High Memory Usage**:
- Reduce cache TTL values
- Monitor cache hit ratios
- Consider cache size limits

**Slow Response Times**:
- Increase connection pool sizes
- Check BigFix server performance
- Monitor network latency

## Security Hardening

### 1. File Permissions
```bash
chmod 600 bem.json                    # Config file
chmod 700 data/                       # Data directory
chmod 644 data/registered_clients.json # Client data
```

### 2. Network Security
- Use HTTPS in production (reverse proxy)
- Restrict access to management ports
- Use VPN for BigFix server connections

### 3. Key Management
- Regular key rotation (yearly)
- Secure storage of configuration files
- Monitor failed authentication attempts

---

This deployment guide provides comprehensive instructions for installing, configuring, and maintaining the BEM server in both development and production environments.