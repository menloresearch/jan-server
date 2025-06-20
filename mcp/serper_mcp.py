import aiohttp
from typing import Dict, Any, Optional
from fastmcp import FastMCP
from config import config

serper_mcp = FastMCP("SerperMCP")


async def make_serper_request(endpoint: str, payload: Dict[str, Any]) -> Dict[str, Any]:
    """Make a request to Serper.dev API"""
    headers = {"X-API-KEY": config.serper_api_key, "Content-Type": "application/json"}

    async with aiohttp.ClientSession() as session:
        async with session.post(
            f"https://google.serper.dev/{endpoint}", headers=headers, json=payload
        ) as response:
            if response.status != 200:
                error_text = await response.text()
                raise Exception(f"Serper API error ({response.status}): {error_text}")

            return await response.json()


@serper_mcp.tool()
async def google_search(
    query: str,
    num_results: int = 10,
    country: str = "us",
    language: str = "en",
) -> Dict[str, Any]:
    """
    Search Google to find more information regarding a topic

    Args:
        query: The search query
        num_results: Number of results to return (1-100, default: 10)
        country: Country code for search (default: us)
        language: Language code for search (default: en)

    Returns:
        Dictionary containing search results
    """
    payload = {
        "q": query,
        "num": min(max(num_results, 1), 100),  # Clamp between 1-100
        "gl": country,
        "hl": language,
    }

    try:
        result = await make_serper_request("search", payload)

        # Format the response for better readability
        formatted_result = {
            "query": query,
            "total_results": result.get("searchInformation", {}).get(
                "totalResults", "Unknown"
            ),
            "organic_results": [],
            "knowledge_graph": result.get("knowledgeGraph"),
            "answer_box": result.get("answerBox"),
            "people_also_ask": result.get("peopleAlsoAsk", []),
        }

        # Process organic results
        for item in result.get("organic", []):
            formatted_result["organic_results"].append(
                {
                    "title": item.get("title", ""),
                    "link": item.get("link", ""),
                    "snippet": item.get("snippet", ""),
                    "position": item.get("position", 0),
                }
            )

        return formatted_result

    except Exception as e:
        return {"error": f"Search failed: {str(e)}", "query": query}


@serper_mcp.tool()
async def read_page(
    url: str,
    include_html: bool = False,
    selector: Optional[str] = None,
) -> Dict[str, Any]:
    """
    Read a webpage by using the provided url

    Args:
        url: The URL to scrape
        include_html: Whether to include raw HTML in response (default: False)
        selector: CSS selector to extract specific elements (optional)

    Returns:
        Dictionary containing webpage content
    """
    payload = {"url": url}

    # Add optional parameters
    if include_html:
        payload["includeHtml"] = True

    if selector:
        payload["selector"] = selector

    try:
        result = await make_serper_request("scrape", payload)

        # Format the response
        formatted_result = {
            "url": url,
            "title": result.get("title", ""),
            "text": result.get("text", ""),
            "status": result.get("status", "unknown"),
        }

        # Add optional fields if present
        if include_html and "html" in result:
            formatted_result["html"] = result["html"]

        if "links" in result:
            formatted_result["links"] = result["links"]

        if "images" in result:
            formatted_result["images"] = result["images"]

        return formatted_result

    except Exception as e:
        return {"error": f"Scraping failed: {str(e)}", "url": url}


@serper_mcp.tool()
async def health_check() -> Dict[str, str]:
    """
    Check if the MCP server and Serper API are working

    Returns:
        Status information
    """
    try:
        # Test with a simple search
        _ = await make_serper_request("search", {"q": "test", "num": 1})
        return {
            "status": "healthy",
            "serper_api": "connected",
            "message": "MCP server is running and Serper API is accessible",
        }
    except Exception as e:
        return {
            "status": "unhealthy",
            "serper_api": "error",
            "message": f"Health check failed: {str(e)}",
        }
