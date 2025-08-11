import json
from datetime import datetime
from typing import Dict, Any

from config import config


def get_current_date():
    return datetime.now().strftime("%B %d, %Y")


async def get_mcp_tools() -> list:
    """Fetch available tools from MCP server"""
    import aiohttp

    try:
        async with aiohttp.ClientSession() as session:
            # First try to get tools via MCP protocol
            mcp_request = {"jsonrpc": "2.0", "id": "tools_list", "method": "tools/list"}

            async with session.post(
                f"http://localhost:{config.port}/mcp/", json=mcp_request
            ) as response:
                if response.status == 200:
                    data = await response.json()
                    if "result" in data and "tools" in data["result"]:
                        tools = data["result"]["tools"]

                        # Convert MCP tools to OpenAI tool format
                        openai_tools = []
                        for tool in tools:
                            openai_tool = {
                                "type": "function",
                                "function": {
                                    "name": tool["name"],
                                    "description": tool["description"],
                                    "parameters": tool.get("inputSchema", {}),
                                },
                            }
                            openai_tools.append(openai_tool)

                        return openai_tools
                    else:
                        return []
                else:
                    return []
    except Exception as e:
        print(f"Error fetching MCP tools: {e}")
        return []


async def call_mcp_tool(tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Call a tool via MCP server"""
    import aiohttp

    mcp_request = {
        "jsonrpc": "2.0",
        "id": f"tool_call_{tool_name}",
        "method": "tools/call",
        "params": {"name": tool_name, "arguments": arguments},
    }

    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"http://localhost:{config.port}/mcp/", json=mcp_request
            ) as response:
                if response.status == 200:
                    data = await response.json()
                    if "result" in data:
                        return data["result"]
                    elif "error" in data:
                        return {"error": data["error"]["message"]}
                else:
                    return {"error": f"HTTP {response.status}"}
    except Exception as e:
        return {"error": str(e)}


def format_tool_result(tool_name: str, result: Dict[str, Any]) -> str:
    """Format tool result for display"""
    if result is None:
        return f"Tool {tool_name} returned no result"

    if "error" in result:
        return f"Error calling {tool_name}: {result['error']}"

    if "content" in result and isinstance(result["content"], list):
        content_parts = []
        for content_item in result["content"]:
            if content_item.get("type") == "text":
                content_parts.append(content_item.get("text", ""))
        return "\n".join(content_parts)

    return json.dumps(result, indent=2)

