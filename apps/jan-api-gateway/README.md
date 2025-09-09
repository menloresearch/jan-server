# Jan API Gateway

A comprehensive API gateway for Jan Server that provides OpenAI-compatible endpoints, multi-tenant organization management, conversation handling, and AI model inference capabilities.

## 🚀 Features

### Core Features
- **OpenAI-Compatible API**: Full compatibility with OpenAI's chat completion API with streaming support
- **Multi-Tenant Architecture**: Organization and project-based access control with hierarchical permissions
- **Conversation Management**: Persistent conversation storage and retrieval with item-level management
- **Authentication & Authorization**: JWT-based auth with Google OAuth2 integration
- **API Key Management**: Secure API key generation and management at organization and project levels
- **Model Registry**: Dynamic model endpoint management with automatic health checking
- **Streaming Support**: Real-time streaming responses with Server-Sent Events (SSE)
- **MCP Integration**: Model Context Protocol support for external tools and resources
- **Web Search**: Serper API integration for web search capabilities via MCP
- **Database Management**: PostgreSQL with read/write replicas and automatic migrations
- **Transaction Management**: Automatic database transaction handling with rollback support
- **Health Monitoring**: Automated health checks with cron-based model endpoint monitoring
- **Performance Profiling**: Built-in pprof endpoints for performance monitoring and Grafana Pyroscope integration
- **Request Logging**: Comprehensive request/response logging with unique request IDs
- **CORS Support**: Cross-origin resource sharing middleware
- **Swagger Documentation**: Auto-generated API documentation with interactive UI



## 🏗️ System Design

![System Design Diagram](docs/System_Design.png)


## 🛠️ Technology Stack

- **Backend**: Go 1.24.6
- **Web Framework**: Gin v1.10.1
- **Database**: PostgreSQL with GORM v1.30.1
- **Database Features**: 
  - Read/Write Replicas with GORM dbresolver
  - Automatic migrations with Atlas
  - Generated query interfaces with GORM Gen
- **Authentication**: JWT v5.3.0 + Google OAuth2 v3.15.0
- **API Documentation**: Swagger/OpenAPI v1.16.6
- **Streaming**: Server-Sent Events (SSE) with chunked transfer
- **Dependency Injection**: Google Wire v0.6.0
- **Logging**: Logrus v1.9.3 with structured logging
- **HTTP Client**: Resty v3.0.0-beta.3
- **Profiling**: 
  - Built-in pprof endpoints
  - Grafana Pyroscope Go integration v0.1.8
- **Scheduling**: Crontab v1.2.0 for health checks
- **MCP Protocol**: MCP-Go v0.37.0 for Model Context Protocol
- **External Integrations**: 
  - Jan Inference Service
  - Serper API (Web Search)
  - Google OAuth2
- **Development Tools**:
  - Atlas for database migrations
  - GORM Gen for code generation
  - Swagger for API documentation

### API Endpoints

#### Authentication (`/jan/v1/auth`)
- `POST /google/callback` - Google OAuth2 callback handler
- `GET /google/testcallback` - Test callback for development

#### Chat Completions (`/jan/v1/chat`)
- `POST /completions` - OpenAI-compatible chat completions with streaming support

#### Conversations (`/jan/v1/conversations`)
- `POST /` - Create new conversation
- `GET /{conversation_id}` - Get conversation by ID
- `PATCH /{conversation_id}` - Update conversation metadata
- `DELETE /{conversation_id}` - Delete conversation
- `POST /{conversation_id}/items` - Add items to conversation
- `GET /{conversation_id}/items` - List conversation items
- `GET /{conversation_id}/items/{item_id}` - Get specific item
- `DELETE /{conversation_id}/items/{item_id}` - Delete specific item

#### Organizations (`/jan/v1/organization`)
- `GET /` - List organizations
- `POST /` - Create organization
- `GET /{org_id}` - Get organization details
- `PATCH /{org_id}` - Update organization
- `DELETE /{org_id}` - Delete organization
- `GET /{org_id}/api_keys` - List organization API keys
- `POST /{org_id}/api_keys` - Create organization API key
- `DELETE /{org_id}/api_keys/{key_id}` - Delete API key

#### Projects (`/jan/v1/organization/{org_id}/projects`)
- `GET /` - List projects
- `POST /` - Create project
- `GET /{project_id}` - Get project details
- `PATCH /{project_id}` - Update project
- `DELETE /{project_id}` - Delete project
- `GET /{project_id}/api_keys` - List project API keys
- `POST /{project_id}/api_keys` - Create project API key
- `DELETE /{project_id}/api_keys/{key_id}` - Delete project API key

#### Models (`/jan/v1/models`)
- `GET /` - List available models

#### MCP (Model Context Protocol) (`/v1/mcp`)
- `POST /mcp` - MCP streamable endpoint with JSON-RPC 2.0 support
- Supported MCP methods:
  - `initialize` - MCP initialization
  - `notifications/initialized` - Initialization notification
  - `ping` - Connection ping
  - `tools/list` - List available tools
  - `tools/call` - Execute tool calls
  - `prompts/list` - List available prompts
  - `prompts/call` - Execute prompts
  - `resources/list` - List available resources
  - `resources/templates/list` - List resource templates
  - `resources/read` - Read resource content
  - `resources/subscribe` - Subscribe to resource updates

#### Health & Version
- `GET /healthcheck` - Health check endpoint
- `GET /v1/version` - API version information
- `GET /google/testcallback` - Development callback test endpoint

## 🚀 Quick Start

### Prerequisites
- Go 1.24.6+
- Docker & Docker Compose
- PostgreSQL (or use Docker)
- Atlas (for database migrations): `brew install ariga/tap/atlas`

### Local Development

1. **Clone and setup**:
   ```bash
   git clone <repository-url>
   cd jan-api-gateway/application
   make setup
   go mod tidy
   ```

2. **Start the server**:
   ```bash
   go run ./cmd/server
   ```

3. **Access the API**:
   - API Base URL: `http://localhost:8080`
   - Swagger UI: `http://localhost:8080/api/swagger/index.html`
   - Health Check: `http://localhost:8080/healthcheck`
   - Version Info: `http://localhost:8080/v1/version`
   - Profiling Endpoints: `http://localhost:6060/debug/pprof/`

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_POSTGRESQL_WRITE_DSN` | Primary database connection | `postgres://jan_user:jan_password@localhost:5432/jan_api_gateway?sslmode=disable` |
| `DB_POSTGRESQL_READ1_DSN` | Read replica database connection | Same as write DSN |
| `JWT_SECRET` | JWT token signing secret | `your-super-secret-jwt-key-change-in-production` |
| `APIKEY_SECRET` | API key encryption secret | `your-api-key-secret-change-in-production` |
| `JAN_INFERENCE_MODEL_URL` | Jan inference service URL | `http://localhost:8000` |
| `SERPER_API_KEY` | Serper API key for web search | `your-serper-api-key` |
| `OAUTH2_GOOGLE_CLIENT_ID` | Google OAuth2 client ID | `your-google-client-id` |
| `OAUTH2_GOOGLE_CLIENT_SECRET` | Google OAuth2 client secret | `your-google-client-secret` |
| `OAUTH2_GOOGLE_REDIRECT_URL` | Google OAuth2 redirect URL | `http://localhost:8080/auth/google/callback` |
| `ALLOWED_CORS_HOSTS` | allowed cors hosts | `http://localhost:8080` |

## 📚 API Usage Examples

### Chat Completion (OpenAI Compatible)

```bash
curl -X POST http://localhost:8080/jan/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "stream": true,
    "temperature": 0.7,
    "max_tokens": 1000
  }'
```

### Create Organization

```bash
curl -X POST http://localhost:8080/jan/v1/organization \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "My Organization",
    "description": "A sample organization"
  }'
```

### Create API Key

```bash
curl -X POST http://localhost:8080/jan/v1/organization/{org_id}/api_keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "My API Key",
    "description": "API key for external integrations"
  }'
```

### Web Search via MCP

```bash
curl -X POST http://localhost:8080/v1/mcp/serper/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "query": "latest AI developments",
    "num_results": 5
  }'
```

## 🔧 Development

### Database Migrations

The project uses Atlas for database migrations. To generate and apply migrations:

1. **Setup migration database**:
   ```sql
   CREATE ROLE migration WITH LOGIN PASSWORD 'migration';
   ALTER ROLE migration WITH SUPERUSER;
   CREATE DATABASE migration WITH OWNER = migration;
   ```

2. **Generate migration files**:
   ```bash
   # Generate schema files
   go run ./cmd/codegen/dbmigration
   
   # Generate diff SQL
   atlas schema diff --dev-url "postgres://migration:migration@localhost:5432/migration?sslmode=disable" \
     --from file://tmp/release.hcl --to file://tmp/main.hcl > tmp/diff.sql
   ```

3. **Apply migrations**:
   ```bash
   # Auto-migration on startup (development)
   go run ./cmd/server
   
   # Manual migration (production)
   atlas migrate apply --url "your-production-db-url"
   ```

### Project Structure

```
jan-api-gateway/
├── application/                 # Main Go application
│   ├── app/
│   │   ├── cmd/server/         # Server entry point
│   │   ├── domain/             # Business logic and entities
│   │   ├── infrastructure/     # Database and external services
│   │   ├── interfaces/         # HTTP handlers and routes
│   │   └── utils/              # Utilities and helpers
│   ├── config/                 # Configuration management
│   ├── docs/                   # Swagger documentation
│   └── Makefile               # Build automation
├── docker/                     # Docker configuration
└── LOCAL_DEV_SETUP.md         # Detailed development setup
```

### Key Features Implementation

#### Streaming with Server-Sent Events
The chat completion endpoint implements real-time streaming using Server-Sent Events (SSE) with chunked transfer encoding, providing low-latency responses for AI model interactions.

#### Multi-Tenant Architecture
Organizations and projects provide hierarchical access control with fine-grained permissions and resource isolation. API keys can be scoped to organization or project levels.

#### OpenAI Compatibility
Full compatibility with OpenAI's chat completion API, including streaming, function calls, tool usage, and all standard parameters (temperature, max_tokens, etc.).

#### Model Context Protocol (MCP)
Comprehensive MCP implementation supporting tools, prompts, and resources with JSON-RPC 2.0 protocol. Includes Serper API integration for web search capabilities.

#### Database Architecture
- Read/Write replica support with automatic load balancing
- Transaction management with automatic rollback on errors
- Generated query interfaces using GORM Gen for type safety
- Automatic schema migrations with Atlas integration

#### Monitoring & Observability
- Built-in pprof endpoints for performance profiling
- Grafana Pyroscope integration for continuous profiling
- Structured logging with unique request IDs
- Automated health checks for inference model endpoints

## 📊 Monitoring & Profiling

### Health Monitoring
- **Health Check Endpoint**: `GET /healthcheck` - Basic server health status
- **Version Endpoint**: `GET /v1/version` - API version information
- **Automated Model Health Checks**: Cron-based monitoring of inference model endpoints
- **Database Health**: Automatic connection monitoring with read/write replica support

### Performance Profiling
- **pprof Endpoints**: Available on port `6060` for performance analysis
  - CPU profiling: `http://localhost:6060/debug/pprof/profile`
  - Memory profiling: `http://localhost:6060/debug/pprof/heap`
  - Goroutine profiling: `http://localhost:6060/debug/pprof/goroutine`
  - Block profiling: `http://localhost:6060/debug/pprof/block`
- **Grafana Pyroscope Integration**: Built-in support for continuous profiling
- **Request Tracing**: Unique request IDs for end-to-end tracing

### Logging
- **Structured Logging**: JSON-formatted logs with Logrus
- **Request/Response Logging**: Complete request lifecycle tracking
- **Error Tracking**: Unique error codes for debugging
- **Streaming Request Handling**: Special handling for SSE and streaming responses

### Database Monitoring
- **Read/Write Replica Support**: Automatic load balancing
- **Connection Pooling**: Optimized database connections
- **Migration Tracking**: Automatic schema migration monitoring
- **Transaction Monitoring**: Automatic rollback on errors


## 📖 Documentation

- **API Documentation**: Available at `/api/swagger/index.html` when running locally
- **Development Setup**: See [LOCAL_DEV_SETUP.md](LOCAL_DEV_SETUP.md) for detailed VS Code/Cursor setup
- **Architecture**: See the mermaid diagram above for system architecture

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request
