import json
from typing import Any, Dict, Optional

import aiohttp
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from logger import logger

from config import config

router = APIRouter(prefix="/mcp", tags=["mcp"])


# MCP Protocol schemas
class MCPRequest(BaseModel):
    jsonrpc: str = "2.0"
    id: Optional[str] = None
    method: str
    params: Optional[Dict[str, Any]] = None


class MCPResponse(BaseModel):
    jsonrpc: str = "2.0"
    id: Optional[str] = None
    result: Optional[Dict[str, Any]] = None
    error: Optional[Dict[str, Any]] = None


class SearchParams(BaseModel):
    q: str
    gl: str = "us"
    hl: str = "en"
    location: Optional[str] = None
    num: Optional[int] = 3
    tbs: Optional[str] = None
    page: Optional[int] = 1
    autocorrect: Optional[bool] = None


class ScrapeParams(BaseModel):
    url: str
    includeMarkdown: Optional[bool] = False


class SerperClient:
    """Client for Serper API integration"""

    def __init__(self, api_key: str):
        self.api_key = api_key
        self.base_url = "https://google.serper.dev"

    async def search(self, params: SearchParams) -> Dict[str, Any]:
        """Perform web search using Serper API"""
        headers = {"X-API-KEY": self.api_key, "Content-Type": "application/json"}

        payload = {
            "q": params.q,
            "gl": params.gl,
            "hl": params.hl,
        }

        # Add optional parameters
        if params.location:
            payload["location"] = params.location
        if params.num:
            payload["num"] = params.num
        if params.tbs:
            payload["tbs"] = params.tbs
        if params.page:
            payload["page"] = params.page
        if params.autocorrect is not None:
            payload["autocorrect"] = params.autocorrect

        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"{self.base_url}/search", headers=headers, json=payload
            ) as response:
                if response.status != 200:
                    error_text = await response.text()
                    raise HTTPException(
                        status_code=response.status,
                        detail=f"Serper API error: {error_text}",
                    )
                return await response.json()

    async def scrape(self, params: ScrapeParams) -> Dict[str, Any]:
        """Scrape webpage content using Serper API"""
        headers = {"X-API-KEY": self.api_key, "Content-Type": "application/json"}

        payload = {"url": params.url, "includeMarkdown": params.includeMarkdown}

        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"{self.base_url}/scrape", headers=headers, json=payload
            ) as response:
                if response.status != 200:
                    error_text = await response.text()
                    raise HTTPException(
                        status_code=response.status,
                        detail=f"Serper API error: {error_text}",
                    )
                return await response.json()


# Initialize Serper client
serper_api_key = config.serper_api_key
logger.info(
    f"Initializing MCP server with SERPER_API_KEY: {'***' if serper_api_key else 'NOT SET'}"
)
if not serper_api_key:
    logger.warning("SERPER_API_KEY not found in environment variables")
    serper_client = None
else:
    serper_client = SerperClient(serper_api_key)


class MCPServer:
    """MCP Server implementation for Serper search functionality"""

    def __init__(self, serper_client: Optional[SerperClient]):
        self.serper_client = serper_client
        self.server_info = {"name": "Serper MCP Server", "version": "0.1.0"}
        self.capabilities = {"tools": {}, "prompts": {}}

    async def handle_request(self, request: MCPRequest) -> MCPResponse:
        """Handle MCP protocol requests"""
        try:
            if request.method == "initialize":
                return MCPResponse(
                    id=request.id,
                    result={
                        "protocolVersion": "2024-11-05",
                        "capabilities": self.capabilities,
                        "serverInfo": self.server_info,
                    },
                )

            elif request.method == "tools/list":
                return MCPResponse(
                    id=request.id,
                    result={
                        "tools": [
                            {
                                "name": "google_search",
                                "description": "Tool to perform web searches via Serper API and retrieve rich results. It is able to retrieve organic search results, people also ask, related searches, and knowledge graph.",
                                "inputSchema": {
                                    "type": "object",
                                    "properties": {
                                        "q": {
                                            "type": "string",
                                            "description": "Search query string",
                                        },
                                        "gl": {
                                            "type": "string",
                                            "description": "Optional region code for search results in ISO 3166-1 alpha-2 format (e.g., 'us')",
                                        },
                                        "hl": {
                                            "type": "string",
                                            "description": "Optional language code for search results in ISO 639-1 format (e.g., 'en')",
                                        },
                                        "location": {
                                            "type": "string",
                                            "description": "Optional location for search results (e.g., 'SoHo, New York, United States', 'California, United States')",
                                        },
                                        "num": {
                                            "type": "number",
                                            "description": "Number of results to return (default: 10)",
                                        },
                                        "page": {
                                            "type": "number",
                                            "description": "Page number of results to return (default: 1)",
                                        },
                                        "tbs": {
                                            "type": "string",
                                            "description": "Time-based search filter ('qdr:h' for past hour, 'qdr:d' for past day, 'qdr:w' for past week, 'qdr:m' for past month, 'qdr:y' for past year)",
                                        },
                                        "autocorrect": {
                                            "type": "boolean",
                                            "description": "Whether to autocorrect spelling in query",
                                        },
                                    },
                                    "required": ["q", "gl", "hl"],
                                },
                            },
                            {
                                "name": "scrape",
                                "description": "Tool to scrape a webpage and retrieve the text and, optionally, the markdown content. It will retrieve also the JSON-LD metadata and the head metadata.",
                                "inputSchema": {
                                    "type": "object",
                                    "properties": {
                                        "url": {
                                            "type": "string",
                                            "description": "The URL of the webpage to scrape.",
                                        },
                                        "includeMarkdown": {
                                            "type": "boolean",
                                            "description": "Whether to include markdown content.",
                                            "default": False,
                                        },
                                    },
                                    "required": ["url"],
                                },
                            },
                        ]
                    },
                )

            elif request.method == "tools/call":
                if not self.serper_client:
                    return MCPResponse(
                        id=request.id,
                        error={
                            "code": -32000,
                            "message": "SERPER_API_KEY not configured",
                        },
                    )

                tool_name = request.params.get("name")
                arguments = request.params.get("arguments", {})

                if tool_name == "google_search":
                    search_params = SearchParams(**arguments)
                    result = await self.serper_client.search(search_params)
                    return MCPResponse(
                        id=request.id,
                        result={
                            "content": [
                                {"type": "text", "text": json.dumps(result, indent=2)}
                            ]
                        },
                    )

                elif tool_name == "scrape":
                    scrape_params = ScrapeParams(**arguments)
                    result = await self.serper_client.scrape(scrape_params)
                    return MCPResponse(
                        id=request.id,
                        result={
                            "content": [
                                {"type": "text", "text": json.dumps(result, indent=2)}
                            ]
                        },
                    )

                else:
                    return MCPResponse(
                        id=request.id,
                        error={"code": -32601, "message": f"Unknown tool: {tool_name}"},
                    )

            else:
                return MCPResponse(
                    id=request.id,
                    error={
                        "code": -32601,
                        "message": f"Unknown method: {request.method}",
                    },
                )

        except Exception as e:
            logger.error(f"MCP request error: {str(e)}")
            return MCPResponse(id=request.id, error={"code": -32000, "message": str(e)})


# Initialize MCP server
mcp_server = MCPServer(serper_client)


@router.post("/")
async def handle_mcp_request(request: MCPRequest):
    """Handle MCP protocol requests"""
    logger.info(f"MCP request: {request.method}")
    response = await mcp_server.handle_request(request)
    return response


@router.get("/capabilities")
async def get_capabilities():
    """Get MCP server capabilities"""
    return {
        "serverInfo": mcp_server.server_info,
        "capabilities": mcp_server.capabilities,
        "tools": [
            {
                "name": "google_search",
                "description": "Perform web searches via Serper API",
            },
            {"name": "scrape", "description": "Scrape webpage content"},
        ],
    }


@router.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "serper_configured": serper_client is not None}
