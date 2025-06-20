from fastmcp import FastMCP
from serper_mcp import serper_mcp
import uvicorn
from config import config

mcp = FastMCP("MenloMCP")
mcp.mount(server=serper_mcp, prefix="serper")

http_app = mcp.http_app(transport="streamable-http")

if __name__ == "__main__":
    uvicorn.run(
        "main:http_app",
        host="0.0.0.0",
        port=config.port,
        # workers=4,
    )
