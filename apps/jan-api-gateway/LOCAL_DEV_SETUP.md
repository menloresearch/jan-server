# Local Development Setup - VS Code/Cursor IDE

This guide will help you set up and run the Jan API Gateway locally using VS Code/Cursor's integrated debugging and launch configurations.

## Prerequisites

- **VS Code** or **Cursor IDE** installed
- **Go extension** for VS Code/Cursor installed
- **Docker and Docker Compose** installed
- **Go 1.19+** installed
- **Git** installed

## Project Structure

```
jan-api-gateway/
├── .vscode/                         # VS Code/Cursor configuration
│   ├── launch.json                 # Debug and launch configurations
│   └── tasks.json                  # Automated tasks (database management)
├── docker/                         # Docker configuration
│   ├── docker-compose.yml         # PostgreSQL service configuration
│   └── init.sql                   # Database initialization script
├── application/                    # Go application code
│   ├── cmd/server/                # Main server entry point
│   ├── app/                       # Application layers
│   └── Makefile                   # Build automation (optional)
└── LOCAL_DEV_SETUP.md             # This documentation
```

## 🚀 Quick Start Guide

### Step 1: Open Project in VS Code/Cursor

1. **Open VS Code/Cursor**
2. **File → Open Folder** → Select the `jan-api-gateway` directory
3. **Install Go extension** if prompted
4. **Trust the workspace** when prompted

### Step 2: Start Development Environment

1. **Press `F5`** or **Run → Start Debugging**
2. **Select "Launch Jan API Gateway (Debug)"** from the dropdown
3. **Wait for automatic setup:**
   - Database starts automatically
   - Environment variables are set
   - Application launches with debugger attached

That's it! Your development environment is ready. 🎉

## 🎯 Available Launch Configurations

### 1. **Launch Jan API Gateway (Debug)** ⭐ *Recommended*
- **Purpose**: Full development environment with debugging
- **What it does**:
  - Automatically starts PostgreSQL database
  - Sets all required environment variables
  - Launches the application with debugger attached
  - Opens integrated terminal for logs
- **When to use**: Daily development and debugging

### 2. **Attach to Jan API Gateway**
- **Purpose**: Attach debugger to already running process
- **What it does**:
  - Connects to a running debug session on port 2345
  - Useful for debugging without restarting the application
- **When to use**: When you want to debug a running instance

### 3. **Launch Tests**
- **Purpose**: Debug unit tests
- **What it does**:
  - Starts database for testing
  - Runs tests with debugging enabled
  - Allows setting breakpoints in test code
- **When to use**: Debugging test failures or test logic

## 🔧 Development Workflow

### Daily Development
1. **Open project** in VS Code/Cursor
2. **Set breakpoints** in your Go code where needed
3. **Press F5** → Select "Launch Jan API Gateway (Debug)"
4. **Code, debug, repeat**:
   - Make code changes
   - Save files (auto-reload on save)
   - Use debug controls to step through code
   - Inspect variables in debug panel

### Debugging Features Available
- ✅ **Breakpoints**: Click left margin to set/remove
- ✅ **Variable Inspection**: Hover over variables or use debug panel
- ✅ **Debug Console**: Execute Go expressions while debugging
- ✅ **Call Stack**: Full call stack visualization
- ✅ **Step Controls**: 
  - `F10` - Step Over
  - `F11` - Step Into 
  - `Shift+F11` - Step Out
  - `F5` - Continue
- ✅ **Watch Expressions**: Monitor specific variables
- ✅ **Conditional Breakpoints**: Right-click breakpoint for conditions

### Testing Workflow
1. **Write your tests** in `*_test.go` files
2. **Set breakpoints** in test code if needed
3. **Press F5** → Select "Launch Tests"
4. **Debug your tests** with full IDE support

## 🛠️ Manual Database Management

While the launch configurations handle the database automatically, you can also manage it manually using VS Code tasks:

### Using Command Palette (Recommended)
1. **Press `Ctrl+Shift+P` (Windows/Linux) or `Cmd+Shift+P` (macOS)**
2. **Type "Tasks: Run Task"**
3. **Select one of:**
   - **Start Database** - Start PostgreSQL
   - **Stop Database** - Stop PostgreSQL
   - **Wait for Database** - Check if database is ready
   - **Build Application** - Build the Go application
   - **Run Tests** - Run all tests

### Using Terminal
```bash
# Start database
docker-compose -f docker/docker-compose.yml up -d postgres

# Stop database
docker-compose -f docker/docker-compose.yml down

# Reset database (removes all data)
docker-compose -f docker/docker-compose.yml down -v
docker-compose -f docker/docker-compose.yml up -d postgres

# View logs
docker-compose -f docker/docker-compose.yml logs postgres

# Connect to database
docker-compose -f docker/docker-compose.yml exec postgres psql -U jan_user -d jan_api_gateway
```

## ⚙️ Environment Variables

The following environment variables are **automatically configured** in the launch configurations:

| Variable | Description | Value |
|----------|-------------|-------|
| `DB_POSTGRESQL_WRITE_DSN` | Primary database connection | `postgres://jan_user:jan_password@localhost:5432/jan_api_gateway?sslmode=disable` |
| `DB_POSTGRESQL_READ1_DSN` | Read replica database connection | `postgres://jan_user:jan_password@localhost:5432/jan_api_gateway?sslmode=disable` |
| `ENABLE_ADMIN_API` | Enable admin API functionality | `True` |
| `JWT_SECRET` | Secret key for JWT token signing | `your-super-secret-jwt-key-change-in-production` |
| `APIKEY_SECRET` | Secret key for API key encryption | `your-api-key-secret-change-in-production` |
| `JAN_INFERENCE_MODEL_URL` | Jan inference model service URL | `http://localhost:8000` |
| `SERPER_API_KEY` | Serper API key for web search | `your-serper-api-key` |
| `OAUTH2_GOOGLE_CLIENT_ID` | Google OAuth2 client ID | `your-google-client-id` |
| `OAUTH2_GOOGLE_CLIENT_SECRET` | Google OAuth2 client secret | `your-google-client-secret` |
| `OAUTH2_GOOGLE_REDIRECT_URL` | Google OAuth2 redirect URL | `http://localhost:8080/auth/google/callback` |

**Note**: You can modify these values in `.vscode/launch.json` if needed for your environment.

## 🐛 Troubleshooting

### Database Connection Issues
1. **Check Docker**: Ensure Docker Desktop is running
2. **Check Port**: Make sure port 5432 is available
3. **View Database Status**: Use Command Palette → "Tasks: Run Task" → "Wait for Database"
4. **View Logs**: Check the integrated terminal for database startup logs

### Go Extension Issues
1. **Install Go Extension**: VS Code/Cursor should prompt you automatically
2. **Go Tools**: Use Command Palette → "Go: Install/Update Tools"
3. **Restart IDE**: Sometimes required after installing tools

### Debug Issues
1. **Check Go Installation**: `go version` in terminal
2. **Install Delve**: Will be automatically installed on first debug run
3. **Check Firewall**: Ensure localhost:2345 is accessible

### Permission Issues
- **Windows**: Run VS Code/Cursor as Administrator if Docker access issues
- **Linux/macOS**: Ensure your user is in the `docker` group

## 🏗️ Database Schema

The application automatically creates and migrates the database schema on startup. The schema includes:

- **Users** - User accounts and authentication
- **Organizations** - Multi-tenant organization structure
- **Projects** - Project management within organizations
- **API Keys** - API authentication and authorization
- **Additional domain tables** - Based on Go structs in the `domain` package

All tables are created automatically using GORM migrations when the application starts.

## 📝 Additional Notes

### Hot Reload
- The debugger supports hot reload - save your Go files and the application will restart automatically
- Breakpoints will be preserved across restarts

### Multiple Debug Sessions
- You can run multiple debug sessions simultaneously
- Use "Attach to Jan API Gateway" to connect additional debuggers

### Production Environment Variables
- For production deployment, replace the example values in environment variables
- Use secure, randomly generated secrets for JWT and API keys
- Configure proper database connections for your production database

### IDE Extensions Recommended
- **Go** - Official Go language support
- **Docker** - Docker container management
- **PostgreSQL** - Database query and management (optional)
- **REST Client** - API testing (optional)

---

**Happy Coding! 🚀** Your Jan API Gateway development environment is now fully integrated with VS Code/Cursor for the best possible developer experience.