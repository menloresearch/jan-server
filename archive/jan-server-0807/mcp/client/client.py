import asyncio
import httpx


async def test():
    async with httpx.AsyncClient() as client:
        url = "http://localhost:4800/mcp/"
        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json, text/event-stream",
            # "Authorization": f"Bearer {token}",
        }

        # # Step 1: Initialize session
        # init_response = await client.post(
        #     url,
        #     json={
        #         "jsonrpc": "2.0",
        #         "id": 1,
        #         "method": "initialize",
        #         "params": {
        #             "protocolVersion": "2024-11-05",
        #             "capabilities": {},
        #             "clientInfo": {"name": "test-client", "version": "1.0.0"},
        #         },
        #     },
        #     headers=headers,
        # )
        #
        # print("Initialize:", init_response.json())
        #
        # # Step 2: Send initialized notification
        # await client.post(
        #     url,
        #     json={"jsonrpc": "2.0", "method": "notifications/initialized"},
        #     headers=headers,
        # )

        # Step 3: Now test tools
        # List tools
        r = await client.post(
            url,
            json={"jsonrpc": "2.0", "id": 2, "method": "tools/list"},
            headers=headers,
        )
        print("Tools:", r.json())

        # Test echo
        r = await client.post(
            url,
            json={
                "jsonrpc": "2.0",
                "id": 3,
                "method": "tools/call",
                "params": {"name": "echo", "arguments": {"text": "Hello!"}},
            },
            headers=headers,
        )
        print("Echo:", r.json())

        r = await client.post(
            url,
            json={
                "jsonrpc": "2.0",
                "id": 3,
                "method": "tools/call",
                "params": {"name": "add", "arguments": {"a": 2, "b": 3}},
            },
            headers=headers,
        )
        print("Echo:", r.json())

        url = "http://localhost:4800/capabilities/"
        r = await client.get(
            url,
            headers=headers,
        )
        print("Echo:", r.json())


if __name__ == "__main__":
    asyncio.run(test())
