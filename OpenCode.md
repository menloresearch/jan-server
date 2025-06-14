# OpenCode Configuration

## Build/Run Commands
- **Run server**: `python server/main.py` or `python -m server.main`
- **Run with venv**: `source .venv/bin/activate && python server/main.py`
- **Run with SERPER_API_KEY**: `SERPER_API_KEY=your_key python server/main.py`
- **Test scripts**: `python test-script/chat-completion.py` (streaming load test), `python test-script/completion.py` (completion test)
- **Single test**: No formal test framework - use individual test scripts in `test-script/` directory

## Code Style Guidelines
- **Imports**: Standard library first, third-party, then local imports (relative imports for same package)
- **Type hints**: Use Pydantic BaseModel for schemas, type hints for function parameters
- **Naming**: snake_case for functions/variables, PascalCase for classes, UPPER_CASE for constants
- **Error handling**: Use FastAPI HTTPException with proper status codes and error objects
- **Logging**: Use custom logger from `logger.py` with structured logging (INFO/DEBUG/ERROR levels)
- **Config**: Environment variables via Pydantic Config class in `config.py` (includes SERPER_API_KEY)
- **Async**: Use async/await for I/O operations, AsyncOpenAI client for LLM calls
- **API**: FastAPI routers with proper prefixes, tags, and documentation
- **Schemas**: Pydantic models in separate schema files for request/response validation
- **Dependencies**: Use FastAPI Depends() for auth, rate limiting via slowapi
- **Streaming**: Use StreamingResponse for SSE with proper headers and generators

## Project Structure
- `server/` - Main application code
- `server/routes/` - API route handlers  
- `server/protocol/` - Protocol implementations (OpenAI compatibility)
- `server/skills/` - Feature modules (deep_research, mcp, research)
- `server/skills/research/` - Intelligent research skill with tool calling
- `server/routes/mcp.py` - Model Context Protocol server for Serper search/scrape
- `test-script/` - Manual testing scripts