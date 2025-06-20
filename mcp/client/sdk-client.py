import asyncio
from mcp.client.session import ClientSession
from mcp.client.streamable_http import streamablehttp_client


async def test():
    # Create client with authentication headers
    token = 12345678
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json",
    }

    server_url = "http://mcp-server.menlo.ai/mcp"

    # Use MCP SDK client
    async with streamablehttp_client(server_url, headers=headers) as (read, write, _):
        async with ClientSession(read, write) as session:
            # Initialize session (handles handshake automatically)
            await session.initialize()

            # List tools
            tools_result = await session.list_tools()
            print("Tools:", tools_result.tools)

            # Call echo tool
            echo_result = await session.call_tool("louis_is_better", {"name": "Yuuki"})
            print("louis_is_better result:", echo_result.content[0].text)


if __name__ == "__main__":
    asyncio.run(test())
