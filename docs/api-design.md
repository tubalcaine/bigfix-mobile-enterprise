# BigFix Mobile Enterprise - API Design

## REST API Overview

The BEM server exposes a REST API that provides access to BigFix server data through a caching proxy layer. The API is designed to be consumed by mobile applications while providing authentication, caching, and format conversion capabilities.

## Base URL Structure

```
https://bem-server:17967/
```

## Authentication

### Client Authentication
**Header**: `Authorization: Client <base64-encoded-private-key>`

```http
Authorization: Client LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQ...
```

### Admin Authentication
**Cookie**: `bem_session=<session-token>`

Sessions expire after 8 hours and are created via the OTP flow.

## API Endpoints

### 1. Registration Endpoints (No Authentication)

#### Generate Registration Request
```http
GET /requestregistration?ClientName=<device-name>
```

**Response**:
```json
{
  "success": true,
  "message": "Registration request created",
  "client_name": "AndroidDevice_22245",
  "instructions": "Place file in registrations/ folder and restart server"
}
```

#### Complete Registration
```http
POST /register
Content-Type: application/json

{
  "ClientName": "AndroidDevice_22245",
  "OneTimeKey": "abc123def456"
}
```

**Success Response**:
```json
{
  "Success": true,
  "Message": "Registration successful",
  "PrivateKey": "-----BEGIN RSA PRIVATE KEY-----\n...",
  "PublicKey": "-----BEGIN PUBLIC KEY-----\n...",
  "ExpiresAt": "2026-08-25T15:13:15Z",
  "KeySize": 4096
}
```

**Error Response**:
```json
{
  "Success": false,
  "Message": "Invalid ClientName or OneTimeKey"
}
```

### 2. Admin Endpoints (No Authentication)

#### Create Admin Session
```http
GET /otp?OneTimeKey=<otp-key>
POST /otp
```

**Response**:
```json
{
  "success": true,
  "message": "Admin session created successfully",
  "expires": "8 hours from now"
}
```

Sets cookie: `bem_session=<token>; HttpOnly; Max-Age=28800`

#### Help
```http
GET /help
```

Returns HTML page listing all available endpoints.

#### Debug (Development Only)
```http
GET /debug/servers
```

**Response**:
```json
{
  "debug": "no-auth-required",
  "ServerNames": ["https://server1:52311", "https://server2:52311"],
  "NumberOfServers": 2,
  "message": "This is a debug endpoint. Remove in production."
}
```

### 3. Protected Endpoints (Authentication Required)

#### Query BigFix Data
```http
GET /urls?url=<encoded-bigfix-url>
POST /urls
Content-Type: application/json

{
  "url": "https://server:52311/api/query?output=json&relevance=names of bes computers"
}
```

**URL Format Examples**:
- Actions: `/api/query?output=json&relevance=(name of it, id of it, state of it) of bes actions`
- Computers: `/api/query?output=json&relevance=names of bes computers`
- Custom Query: `/api/query?relevance=<relevance-expression>`

**Success Response** (JSON Format):
```json
{
  "cacheitem": {
    "result": [
      ["Computer1", 1001],
      ["Computer2", 1002]
    ],
    "plural": true,
    "type": "( string, integer )"
  }
}
```

**Success Response** (XML Converted):
```json
{
  "cacheitem": {
    "action": [
      {
        "resource": "https://server:52311/api/action/123",
        "name": "Update Software",
        "id": "123"
      }
    ]
  }
}
```

**Error Response**:
```json
{
  "cacheitem": "",
  "error": "server cache does not exist for https://server:52311"
}
```

#### List Configured Servers
```http
GET /servers
POST /servers
```

**Response**:
```json
{
  "ServerNames": [
    "https://server1:52311",
    "https://server2:52311"
  ],
  "NumberOfServers": 2
}
```

#### Cache Summary
```http
GET /summary
POST /summary
```

**Response**:
```json
{
  "https://server1:52311": {
    "total_items": 150,
    "expired_items": 5,
    "current_items": 145,
    "serverSize": 2048576
  },
  "https://server2:52311": {
    "total_items": 200,
    "expired_items": 10,
    "current_items": 190,
    "serverSize": 3145728
  },
  "totalSize": 5194304
}
```

#### Cache Contents
```http
GET /cache
POST /cache
```

**Response**:
```json
{
  "https://server1:52311": [
    "https://server1:52311/api/actions",
    "https://server1:52311/api/computers",
    "https://server1:52311/api/query?relevance=names of bes computers"
  ],
  "https://server2:52311": [
    "https://server2:52311/api/sites"
  ]
}
```

## Request Processing Flow

### 1. Format Detection

The API automatically detects the requested output format based on URL parameters:

**JSON Passthrough** (when `output=json` or `format=json`):
- Request sent directly to BigFix server
- JSON response passed through without conversion
- Optimal for mobile apps expecting BigFix's native JSON format

**XML Conversion** (default or `format=xml`):
- Request sent to BigFix server (XML format)
- XML parsed into Go structs (BESAPI/BES)
- Converted to structured JSON for client

### 2. URL Encoding

All query parameters are properly URL-encoded before forwarding to BigFix servers:

```
Input:  /api/query?relevance=(name of it, id of it) of bes actions
Output: /api/query?relevance=%28name+of+it%2C+id+of+it%29+of+bes+actions
```

### 3. Caching Strategy

- **Cache Key**: Full URL including query parameters
- **TTL**: Configurable per server (default 300 seconds)
- **Thread Safety**: Concurrent read/write operations supported
- **Expiration**: Lazy expiration on access

### 4. Authentication Flow

```
1. Extract Authorization header or session cookie
2. Validate RSA private key or session token  
3. Check key expiration and registration status
4. Update last-used timestamp
5. Process request or return 401
```

## Error Handling

### HTTP Status Codes

- **200 OK**: Successful request with data
- **401 Unauthorized**: Authentication required or failed
- **404 Not Found**: Cache miss or server not configured
- **409 Conflict**: Client already registered
- **500 Internal Server Error**: Server processing error

### Error Response Format

```json
{
  "error": "Authentication required. Please visit /otp?OneTimeKey=<key> or register your client.",
  "expired": false
}
```

**Special Error Fields**:
- `expired: true`: Signals mobile app to discard keys and re-register

### Common Error Scenarios

1. **Expired Keys**:
   ```json
   {
     "error": "Client authentication failed. Key may be expired or invalid.",
     "expired": true
   }
   ```

2. **Server Not Configured**:
   ```json
   {
     "cacheitem": "",
     "error": "server cache does not exist for https://unknown-server:52311"
   }
   ```

3. **Invalid BigFix Query**:
   ```json
   {
     "cacheitem": "",
     "error": "EOF"
   }
   ```

## BigFix Query Examples

### Common Actions Query
```
/api/query?output=json&relevance=(name of it, id of it, state of it, number of results of it, name of issuer of it, multiple flag of it) of bes actions
```

### Computer Information
```
/api/query?output=json&relevance=(name of it, id of it, last report time of it) of bes computers
```

### Site Listing  
```
/api/query?output=json&relevance=(name of it, type of it) of bes sites
```

### Custom Properties
```
/api/query?output=json&relevance=names of properties of type "retrieved" of bes computers
```

## Response Format Comparison

### BigFix Native JSON (output=json)
```json
{
  "cacheitem": {
    "result": [["Action1", 123, "Open"]],
    "plural": true,
    "type": "( string, integer, string )"
  }
}
```

### BEM XML Conversion
```json
{
  "cacheitem": {
    "query": [{
      "resource": "https://server/api/query?relevance=...",
      "result": {
        "answer": [
          {"type": "string", "value": "Action1"},
          {"type": "integer", "value": "123"},
          {"type": "string", "value": "Open"}
        ]
      }
    }]
  }
}
```

## Rate Limiting & Performance

### Connection Pooling
- Configurable pool size per BigFix server
- 30-second connection acquisition timeout
- HTTP/1.1 persistent connections

### Caching Performance  
- Memory-based caching with TTL
- Concurrent access via sync.Map
- Background cache population supported

### Request Timeouts
- Default HTTP client timeout: 120 seconds
- Connection pool acquisition: 30 seconds
- Session cookie: 8 hours

---

This API design provides a robust interface for mobile applications to access BigFix data while maintaining security, performance, and compatibility with both BigFix's native JSON format and structured XML conversion.