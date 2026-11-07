# Developers Guide

## Swagger UI

### View REST API Specification from Swagger Editor UI

Open https://editor.swagger.io/ in a browser. Click `File` - `Import file` from top menu and open the file [$WORKSPACE/redhat-cne/rest-api/v2/swagger.json](../v2/swagger.json). This loads the REST API specification like the following screenshot:

![Alt text](swagger-editor.png "Swagger Editor")

The updated Swagger specification includes:

- **Authentication Documentation**: Comprehensive mTLS and OAuth 2.0 security definitions
- **Enhanced API Descriptions**: Detailed endpoint descriptions with authentication requirements
- **Security Schemes**: Proper documentation of dual authentication (mTLS + OAuth)
- **Error Responses**: Complete 401 Unauthorized responses for protected endpoints
- **Tags and Categories**: Organized endpoints by functionality (Subscriptions, Publishers, Events, HealthCheck, Authentication)

### Interact with REST-API in Swagger UI

You can interact with API endpoint by click `Try it out`, enter required parameters and click `Execute`.
This requires a REST-API server to be deployed at backend and accessible from localhost.

**Important**: When testing authenticated endpoints, you must:

1. **Configure mTLS**: Set up client certificates in your HTTP client
2. **Provide OAuth Token**: Include valid Bearer token in Authorization header
3. **Use HTTPS**: Ensure secure connection for mTLS authentication

Example authentication setup:
```bash
# For mTLS
--cert /path/to/client.crt \
--key /path/to/client.key \
--cacert /path/to/ca.crt \

# For OAuth
-H "Authorization: Bearer your_jwt_token_here"
```

## Generate Swagger Spec

The swagger documentation of this repo is generated using tools and annotations based on https://github.com/go-swagger/go-swagger. The current version of go-swagger has an issue of generating empty definitions with go 1.20+. The workaround is to run swagger tool from docker.

Use the following commands to generate swagger spec file [v2/swagger.json](../v2/swagger.json).

```sh
go install github.com/go-swagger/go-swagger/cmd/swagger@latest
alias swagger='docker run --rm -it  --user $(id -u):$(id -g) -v $HOME:$HOME -w $PWD quay.io/goswagger/swagger'
swagger version

# generate spec without Go language specific extensions
cd $WORKSPACE/redhat-cne/rest-api/v2
SWAGGER_GENERATE_EXTENSION=false swagger generate spec --input tags.json -o swagger.json

# validate spec
swagger validate swagger.json
```

**Note**: The swagger.json file has been enhanced with:
- Security definitions for mTLS and OAuth 2.0
- Authentication requirements for protected endpoints
- Comprehensive error response documentation
- Updated API descriptions and metadata

## Generate REST API Documentation

Use the following commands to generate swagger documentation markdown file [rest_api_v2.md](rest_api_v2.md).

```sh
# generate markdown doc
cd $WORKSPACE/redhat-cne/rest-api/v2
swagger generate markdown --skip-validation --output=../docs/rest_api_v2.md
```

The generated documentation includes:

- **Security Model**: Complete authentication and authorization documentation
- **Endpoint Reference**: All endpoints with authentication requirements
- **Request/Response Examples**: Sample payloads and responses
- **Error Handling**: Comprehensive error response documentation
- **Authentication Guide**: mTLS and OAuth integration examples

## Authentication Testing

When testing the API with authentication enabled:

### mTLS Testing
```bash
curl -X POST https://localhost:9043/api/ocloudNotifications/v2/subscriptions \
  --cert /path/to/client.crt \
  --key /path/to/client.key \
  --cacert /path/to/ca.crt \
  -H "Content-Type: application/json" \
  -d '{"EndpointUri": "http://example.com/callback", "ResourceAddress": "/test/resource"}'
```

### OAuth Testing
```bash
curl -X POST https://localhost:9043/api/ocloudNotifications/v2/subscriptions \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"EndpointUri": "http://example.com/callback", "ResourceAddress": "/test/resource"}'
```

### Dual Authentication Testing
```bash
curl -X POST https://localhost:9043/api/ocloudNotifications/v2/subscriptions \
  --cert /path/to/client.crt \
  --key /path/to/client.key \
  --cacert /path/to/ca.crt \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"EndpointUri": "http://example.com/callback", "ResourceAddress": "/test/resource"}'
```
