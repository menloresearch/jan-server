import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from routes import api, model
from routes.limiter import limiter
from slowapi import _rate_limit_exceeded_handler
from slowapi.errors import RateLimitExceeded
from dotenv import load_dotenv
import os


load_dotenv()

PORT = os.getenv("PORT")

app = FastAPI(title="vLLM Proxy with API Key Auth")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # TODO: Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(api.router)
app.include_router(model.router)

app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)

# params = StdioServerParameters(
#     command="uvx",
#     args=["mcp-server-fetch"],
# )


# async def start_mcp():
#     async with stdio_client(params) as streams:
#         async with ClientSession(streams[0], streams[1]) as session:
#             await session.initialize()
#             print(session)
#             await session
#             print(session)


if __name__ == "__main__":
    # asyncio.run(start_mcp())
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=int(PORT),
    )
