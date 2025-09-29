# Jan API Gateway

A comprehensive API gateway for Jan Server that provides OpenAI-compatible endpoints, multi-tenant organization management, conversation handling, and AI model inference capabilities. The system serves as a centralized gateway for AI model interactions with enterprise-grade features including user management, organization hierarchies, project-based access control, and real-time streaming responses.

## 🚀 Features

### Core Features
- **OpenAI-Compatible API**: Full compatibility with OpenAI's chat completion API with streaming support and reasoning content handling
- **Multi-Tenant Architecture**: Organization and project-based access control with hierarchical permissions and member management
- **Conversation Management**: Persistent conversation storage and retrieval with item-level management, including message, function call, and reasoning content types
- **Authentication & Authorization**: JWT-based auth with Google OAuth2 integration and role-based access control
- **API Key Management**: Secure API key generation and management at organization and project levels with multiple key types (admin, project, organization, service, ephemeral)
- **Model Registry**: Dynamic model endpoint management with automatic health checking and service discovery
- **Cache Service**: High-performance caching for inference models using Redis to reduce load times and improve response performance
- **Streaming Support**: Real-time streaming responses with Server-Sent Events (SSE) and chunked transfer encoding
- **MCP Integration**: Model Context Protocol support for external tools and resources with JSON-RPC 2.0
- **Web Search**: Serper API integration for web search capabilities via MCP with webpage fetching
- **Database Management**: PostgreSQL with read/write replicas and automatic migrations using Atlas
- **Transaction Management**: Automatic database transaction handling with rollback support
- **Health Monitoring**: Automated health checks with cron-based model endpoint monitoring
- **Performance Profiling**: Built-in pprof endpoints for performance monitoring and Grafana Pyroscope integration
- **Request Logging**: Comprehensive request/response logging with unique request IDs and structured logging
- **CORS Support**: Cross-origin resource sharing middleware with configurable allowed hosts
- **Swagger Documentation**: Auto-generated API documentation with interactive UI
- **Email Integration**: SMTP support for invitation and notification systems
- **Response Management**: Comprehensive response tracking with status management and usage statistics



## 🏗️ Business Domain Architecture

### Core Domain Models
![System Design Diagram](docs/System_Design.png)
#### User Management
- **Users**: Support for both regular users and guest users with email-based authentication
- **Organizations**: Multi-tenant organizations with owner/member roles and hierarchical access
- **Projects**: Project-based resource isolation within organizations with member management
- **Invites**: Email-based invitation system for organization and project membership

#### Authentication & Authorization
- **API Keys**: Multiple types (admin, project, organization, service, ephemeral) with scoped permissions
- **JWT Tokens**: Stateless authentication with Google OAuth2 integration
- **Role-Based Access**: Hierarchical permissions from organization owners to project members

#### Conversation Management
- **Conversations**: Persistent chat sessions with metadata and privacy controls
- **Items**: Rich conversation items supporting messages, function calls, and reasoning content
- **Content Types**: Support for text, images, files, and multimodal content with annotations
- **Status Tracking**: Real-time status management (pending, in_progress, completed, failed, cancelled)

#### Response Management
- **Responses**: Comprehensive tracking of AI model interactions with full parameter logging
- **Streaming**: Real-time streaming with Server-Sent Events and chunked transfer encoding
- **Usage Statistics**: Token usage tracking and performance metrics
- **Error Handling**: Detailed error tracking with unique error codes

#### External Integrations
- **Jan Inference Service**: Primary AI model inference backend with health monitoring
- **Serper API**: Web search capabilities via MCP with search and webpage fetching
- **SMTP**: Email notifications for invitations and system alerts
- **Model Registry**: Dynamic model discovery and health checking

### Data Flow Architecture

1. **Request Processing**: HTTP requests → Authentication → Authorization → Business Logic
2. **AI Inference**: Request → Jan Inference Service → Streaming Response → Database Storage
3. **MCP Integration**: JSON-RPC 2.0 → Tool Execution → External APIs → Response Streaming
4. **Health Monitoring**: Cron Jobs → Service Discovery → Model Registry Updates
5. **Database Operations**: Read/Write Replicas → Transaction Management → Automatic Migrations


## 🛠️ Technology Stack

- **Backend**: Go 1.24.6
- **Web Framework**: Gin v1.10.1
- **Database**: PostgreSQL with GORM v1.30.1
- **Database Features**: 
  - Read/Write Replicas with GORM dbresolver
  - Automatic migrations with Atlas
  - Generated query interfaces with GORM Gen
- **Authentication**: JWT v5.3.0 + Google OAuth2 v3.15.0
- **Caching**: Redis v9.14.0 for high-performance model caching
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

#### Authentication API (`/v1/auth`)
- `POST /google/callback` - Google OAuth2 callback handler
- `GET /google/testcallback` - Test callback for development

#### Chat Completions API (`/v1/chat`, `/v1/mcp`, `/v1/models`)
- `POST /chat/completions` - OpenAI-compatible chat completions with streaming support
- `POST /mcp` - MCP streamable endpoint with JSON-RPC 2.0 support
- `GET /models` - List available models from inference registry
- Supported MCP methods:
  - `initialize` - MCP initialization
  - `notifications/initialized` - Initialization notification
  - `ping` - Connection ping
  - `tools/list` - List available tools (Serper search, webpage fetch)
  - `tools/call` - Execute tool calls
  - `prompts/list` - List available prompts
  - `prompts/call` - Execute prompts
  - `resources/list` - List available resources
  - `resources/templates/list` - List resource templates
  - `resources/read` - Read resource content
  - `resources/subscribe` - Subscribe to resource updates

#### Conversation-aware Chat API (`/v1/conv`)
- `POST /chat/completions` - Conversation-based chat completions with streaming support
- `POST /mcp` - MCP streamable endpoint for conversation-aware chat
- `GET /models` - List available models for conversation-aware chat

#### Conversations API (`/v1/conversations`)
- `POST /` - Create new conversation
- `GET /` - List conversations with pagination
- `GET /{conversation_id}` - Get conversation by ID
- `PATCH /{conversation_id}` - Update conversation metadata
- `DELETE /{conversation_id}` - Delete conversation
- `POST /{conversation_id}/items` - Add items to conversation
- `GET /{conversation_id}/items` - List conversation items
- `GET /{conversation_id}/items/{item_id}` - Get specific item
- `DELETE /{conversation_id}/items/{item_id}` - Delete specific item

#### Administration API (`/v1/organization`)
- `GET /` - List organizations
- `POST /` - Create organization
- `GET /{org_id}` - Get organization details
- `PATCH /{org_id}` - Update organization
- `DELETE /{org_id}` - Delete organization
- `GET /{org_id}/api_keys` - List organization API keys
- `POST /{org_id}/api_keys` - Create organization API key
- `DELETE /{org_id}/api_keys/{key_id}` - Delete API key
- `GET /admin_api_keys` - List admin API keys
- `POST /admin_api_keys` - Create admin API key
- `GET /admin_api_keys/{key_id}` - Get admin API key
- `DELETE /admin_api_keys/{key_id}` - Delete admin API key

##### Projects (`/v1/organization/{org_id}/projects`)
- `GET /` - List projects
- `POST /` - Create project
- `GET /{project_id}` - Get project details
- `PATCH /{project_id}` - Update project
- `DELETE /{project_id}` - Delete project
- `GET /{project_id}/api_keys` - List project API keys
- `POST /{project_id}/api_keys` - Create project API key
- `DELETE /{project_id}/api_keys/{key_id}` - Delete project API key

##### Invites (`/v1/organization/{org_id}/invites`)
- `GET /` - List organization invites
- `POST /` - Create organization invite
- `GET /{invite_id}` - Get invite details
- `DELETE /{invite_id}` - Delete invite

#### Responses API (`/v1/responses`)
- `POST /` - Create response
- `GET /{response_id}` - Get response details
- `DELETE /{response_id}` - Delete response
- `POST /{response_id}/cancel` - Cancel running response
- `GET /{response_id}/input_items` - List response input items

#### Server API (`/v1/version`, `/healthcheck`)
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
| `ALLOWED_CORS_HOSTS` | Value of allowed CORS hosts, separated by commas, supporting prefix wildcards with '*'. | `http://localhost:8080,*jan.ai` |
| `SMTP_HOST` | SMTP server host for email notifications | `smtp.gmail.com` |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USERNAME` | SMTP username | `your-smtp-username` |
| `SMTP_PASSWORD` | SMTP password | `your-smtp-password` |
| `SMTP_SENDER_EMAIL` | Default sender email address | `noreply@yourdomain.com` |
| `INVITE_REDIRECT_URL` | Redirect URL for invitation acceptance | `http://localhost:8080/invite/accept` |
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379` |
| `REDIS_PASSWORD` | Redis authentication password | `` (empty for dev) |
| `REDIS_DB` | Redis database number | `0` |

## 🚀 Redis Caching

The Jan API Gateway includes Redis caching for inference models to significantly improve performance by avoiding repeated model loading and caching identical requests.

### Redis Features
- **Model List Caching**: Cache model discovery for 10 minutes
- **Transparent Integration**: No code changes needed in existing handlers
- **Centralized Constants**: Redis cache keys defined as constants

### Quick Setup

1. **Deploy Redis Infrastructure**:
   ```bash
   helm dependency update charts/umbrella-chart/
   helm install jan-server charts/umbrella-chart/
   ```

2. **Environment Variables**:
   ```bash
   REDIS_URL=redis://jan-server-redis-master:6379
   REDIS_PASSWORD=""  # Empty for dev
   REDIS_DB=0
   ```

3. **Verify Setup**:
   ```bash
   # Check Redis connectivity in logs
   kubectl logs deployment/jan-server-jan-api-gateway | grep "Successfully connected to Redis"
   ```

### Performance Benefits
- **Reduced latency** for model discovery calls
- **Reduced CPU usage** by avoiding repeated model loading
- **Better scalability** with reduced backend load
- **Improved user experience** with faster response times

## 📚 API Usage Examples

### Chat Completion (OpenAI Compatible)

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
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

### Conversation-based Chat Completion

```bash
curl -X POST http://localhost:8080/v1/conv/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "jan-v1-4b",
    "input": "Hello, how are you?",
    "conversation_id": "conv_abc123",
    "stream": true,
    "temperature": 0.7,
    "max_tokens": 1000
  }'
```

### Create Organization

```bash
curl -X POST http://localhost:8080/v1/organization \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "My Organization",
    "description": "A sample organization"
  }'
```

### Create API Key

```bash
curl -X POST http://localhost:8080/v1/organization/{org_id}/api_keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "My API Key",
    "description": "API key for external integrations"
  }'
```

### Web Search via MCP

```bash
curl -X POST http://localhost:8080/v1/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "serper_search",
      "arguments": {
        "q": "latest AI developments",
        "num": 5
      }
    }
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
The chat completion endpoints implement real-time streaming using Server-Sent Events (SSE) with chunked transfer encoding, providing low-latency responses for AI model interactions. The system supports both content and reasoning content streaming with proper buffering and event sequencing.

#### Multi-Tenant Architecture
Organizations and projects provide hierarchical access control with fine-grained permissions and resource isolation. API keys can be scoped to organization or project levels with different types (admin, project, organization, service, ephemeral) for various use cases.

#### OpenAI Compatibility
Full compatibility with OpenAI's chat completion API, including streaming, function calls, tool usage, and all standard parameters (temperature, max_tokens, etc.). The system also supports reasoning content and multimodal inputs.

#### Model Context Protocol (MCP)
Comprehensive MCP implementation supporting tools, prompts, and resources with JSON-RPC 2.0 protocol. Includes Serper API integration for web search capabilities and webpage fetching functionality.

#### Database Architecture
- Read/Write replica support with automatic load balancing using GORM dbresolver
- Transaction management with automatic rollback on errors
- Generated query interfaces using GORM Gen for type safety
- Automatic schema migrations with Atlas integration
- Support for complex data types including JSON fields and relationships

#### Monitoring & Observability
- Built-in pprof endpoints for performance profiling on port 6060
- Grafana Pyroscope integration for continuous profiling
- Structured logging with unique request IDs and comprehensive request/response tracking
- Automated health checks for inference model endpoints with cron-based monitoring
- Model registry with dynamic service discovery and health status tracking

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
- **OpenAI-Style Documentation**: Professional API reference documentation with OpenAI-style layout
- **Development Setup**: See [LOCAL_DEV_SETUP.md](LOCAL_DEV_SETUP.md) for detailed VS Code/Cursor setup
- **Architecture**: See the mermaid diagram above for system architecture

### API Structure Overview

The API is organized into the following main groups:

1. **Authentication API** - User authentication and authorization
2. **Chat Completions API** - Chat completions, models, and MCP functionality
3. **Conversation-aware Chat API** - Conversation-based chat completions
4. **Conversations API** - Conversation management and items
5. **Responses API** - Response tracking and management
6. **Administration API** - Organization and project management
7. **Server API** - System information and health checks

### Swagger Documentation

The API documentation is automatically generated from code annotations and includes:
- Interactive API explorer
- Request/response examples
- Authentication requirements
- Error code documentation
- Model schemas and validation rules

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request
