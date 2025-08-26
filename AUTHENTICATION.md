# Authentication Configuration for REST API

This document describes how to configure mTLS (mutual TLS) and OAuth authentication for the REST API server.

## Overview

The REST API supports two authentication mechanisms that can be applied to specific endpoints:

1. **mTLS (Mutual TLS)**: Client certificate-based authentication
2. **OAuth**: Bearer token-based authentication using JWT tokens

Both mechanisms can be enabled independently or together for enhanced security.

## Protected vs Public Endpoints

### Protected Endpoints (Require Authentication)

The following endpoints require authentication when enabled:

#### Subscription Management
- `POST /subscriptions` - Create subscription
- `DELETE /subscriptions/{subscriptionId}` - Delete specific subscription
- `DELETE /subscriptions` - Delete all subscriptions
- `PUT /subscriptions/status/{subscriptionId}` - Ping for subscription status

#### Publisher Management
- `POST /publishers` - Create publisher
- `DELETE /publishers/{publisherid}` - Delete specific publisher
- `DELETE /publishers` - Delete all publishers

#### Event Management
- `POST /create/event` - Publish event
- `POST /log` - Log event

#### Test Endpoints
- `POST /dummy` - Test endpoint
- `POST /dummy2` - Test endpoint

### Public Endpoints (No Authentication Required)

These endpoints remain accessible without authentication:

#### Read Operations
- `GET /subscriptions` - List all subscriptions
- `GET /subscriptions/{subscriptionId}` - Get subscription details
- `GET /publishers` - List all publishers
- `GET /publishers/{publisherid}` - Get publisher details
- `GET /{ResourceAddress}/CurrentState` - Get current state

### Health Endpoint Behavior

The `/health` endpoint has special behavior based on authentication configuration:

#### When Authentication is Disabled
- Accessible via HTTP without any authentication
- Simple health check for service availability

#### When mTLS is Enabled
- Accessible via HTTPS only
- **Requires a valid client certificate** for access
- Used by internal services (like PTP daemon) for health checks
- Service CA certificate is required for internal health checks

**Note**: Even though the `/health` endpoint is considered "public" in terms of business logic, when mTLS is enabled, it still requires proper certificate authentication for security reasons.

## Server Architecture

### Single Server with Conditional Authentication

The REST API uses a single server architecture that adapts based on authentication configuration:

1. **No Authentication**: Server runs on HTTP, all endpoints accessible without authentication
2. **mTLS Only**: Server runs on HTTPS with client certificate validation
3. **OAuth Only**: Server runs on HTTP with Bearer token validation
4. **Both mTLS and OAuth**: Server runs on HTTPS with both client certificate and Bearer token validation

### Health Endpoint Implementation

The `/health` endpoint is always included in the main server but behaves differently based on authentication:

- **Without mTLS**: Accessible via HTTP without authentication
- **With mTLS**: Accessible via HTTPS but requires valid client certificate
- **Internal health checks** (like PTP daemon) use the service CA certificate for authentication

This approach ensures:
- Consistent server architecture
- No port conflicts
- Proper security when mTLS is enabled
- Internal services can still perform health checks

## Configuration

### Authentication Configuration Structure

```go
type AuthConfig struct {
    // mTLS configuration
    EnableMTLS     bool     `json:"enableMTLS"`
    CACertPath     string   `json:"caCertPath"`
    ServerCertPath string   `json:"serverCertPath"`
    ServerKeyPath  string   `json:"serverKeyPath"`

    // OAuth configuration
    EnableOAuth      bool     `json:"enableOAuth"`
    OAuthIssuer      string   `json:"oauthIssuer"`
    OAuthJWKSURL     string   `json:"oauthJWKSURL"`
    RequiredScopes   []string `json:"requiredScopes"`
    RequiredAudience string   `json:"requiredAudience"`
}
```

### Example Configuration

See `auth-config-example.json` for a complete configuration example:

```json
{
  "enableMTLS": true,
  "caCertPath": "/etc/certs/ca.crt",
  "serverCertPath": "/etc/certs/server.crt",
  "serverKeyPath": "/etc/certs/server.key",

  "enableOAuth": true,
  "oauthIssuer": "https://your-oauth-provider.com",
  "oauthJWKSURL": "https://your-oauth-provider.com/.well-known/jwks.json",
  "requiredScopes": ["subscription:create", "events:read"],
  "requiredAudience": "rest-api-service"
}
```

## mTLS Configuration

### Certificate Requirements

1. **CA Certificate** (`caCertPath`): The Certificate Authority certificate used to validate client certificates
2. **Server Certificate** (`serverCertPath`): The server's TLS certificate
3. **Server Private Key** (`serverKeyPath`): The server's private key

### Certificate Generation Example

```bash
# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate CA certificate
openssl req -new -x509 -key ca.key -sha256 -subj "/C=US/ST=CA/O=MyOrg/CN=MyCA" -days 3650 -out ca.crt

# Generate server private key
openssl genrsa -out server.key 4096

# Generate server certificate signing request
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=CA/O=MyOrg/CN=localhost"

# Generate server certificate signed by CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -sha256

# Generate client private key
openssl genrsa -out client.key 4096

# Generate client certificate signing request
openssl req -new -key client.key -out client.csr -subj "/C=US/ST=CA/O=MyOrg/CN=client"

# Generate client certificate signed by CA
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365 -sha256
```

## Client Examples

### Protected Endpoint Examples

#### Create Subscription (with both mTLS and OAuth)

```bash
# With both mTLS and OAuth
curl -X POST https://localhost:9043/api/ocloudNotifications/v2/subscriptions \
  --cert client.crt \
  --key client.key \
  --cacert ca.crt \
  -H "Authorization: Bearer valid_your_jwt_token_here" \
  -H "Content-Type: application/json" \
  -d '{"EndpointUri": "http://example.com/callback", "ResourceAddress": "/test/resource"}'

# With only mTLS (if OAuth is disabled)
curl -X POST https://localhost:9043/api/ocloudNotifications/v2/subscriptions \
  --cert client.crt \
  --key client.key \
  --cacert ca.crt \
  -H "Content-Type: application/json" \
  -d '{"EndpointUri": "http://example.com/callback", "ResourceAddress": "/test/resource"}'

# With only OAuth (if mTLS is disabled)
curl -X POST http://localhost:9043/api/ocloudNotifications/v2/subscriptions \
  -H "Authorization: Bearer valid_your_jwt_token_here" \
  -H "Content-Type: application/json" \
  -d '{"EndpointUri": "http://example.com/callback", "ResourceAddress": "/test/resource"}'
```

#### Delete Publisher (with both mTLS and OAuth)

```bash
curl -X DELETE https://localhost:9043/api/ocloudNotifications/v2/publishers/publisher-id \
  --cert client.crt \
  --key client.key \
  --cacert ca.crt \
  -H "Authorization: Bearer valid_your_jwt_token_here"
```

### Public Endpoint Examples

#### List Subscriptions (no authentication required)

```bash
# Over HTTPS (when mTLS is enabled)
curl -X GET https://localhost:9043/api/ocloudNotifications/v2/subscriptions \
  --cacert ca.crt

# Over HTTP (when mTLS is disabled)
curl -X GET http://localhost:9043/api/ocloudNotifications/v2/subscriptions
```

#### Health Check

```bash
# When mTLS is enabled (requires client certificate)
curl -X GET https://localhost:9043/api/ocloudNotifications/v2/health \
  --cert client.crt \
  --key client.key \
  --cacert ca.crt

# When mTLS is disabled (no authentication required)
curl -X GET http://localhost:9043/api/ocloudNotifications/v2/health

# Internal health check (for services like PTP daemon)
curl -X GET https://localhost:9043/api/ocloudNotifications/v2/health \
  --cacert /etc/cloud-event-proxy/ca-bundle/service-ca.crt
```

## Security Considerations

1. **Certificate Management**
   - Implement proper certificate rotation
   - Use secure storage for private keys
   - Consider using a certificate manager in production

2. **OAuth Security**
   - Use a proper JWT validation library in production
   - Implement token caching and JWKS key rotation
   - Validate all claims (issuer, audience, scopes, expiration)

3. **TLS Configuration**
   - Use TLS 1.2 or higher
   - Configure secure cipher suites
   - Enable HTTP/2 when possible

4. **Access Control**
   - Monitor and log authentication failures
   - Implement rate limiting
   - Consider IP whitelisting for sensitive endpoints

5. **Error Handling**
   - Use generic error messages in production
   - Don't expose internal details in error responses
   - Log detailed errors server-side

6. **Health Endpoint Security**
   - When mTLS is enabled, health endpoint requires client certificates
   - Internal services should use service CA certificates for health checks
   - External health checks require proper client certificates
   - Consider network policies to restrict health endpoint access

## Production Recommendations

1. **Authentication Infrastructure**
   - Use a proper OAuth 2.0 server (e.g., Keycloak, Auth0)
   - Implement a certificate management solution
   - Consider using a service mesh for mTLS

2. **Monitoring and Logging**
   - Log all authentication events
   - Monitor authentication failures
   - Set up alerts for suspicious activity

3. **Security Hardening**
   - Use hardware security modules (HSMs) for key storage
   - Implement certificate revocation checking
   - Regular security audits and penetration testing

4. **Performance Optimization**
   - Implement token caching
   - Use connection pooling
   - Configure appropriate timeouts

5. **Operational Considerations**
   - Document certificate rotation procedures
   - Create incident response plans
   - Regular security training for team members
