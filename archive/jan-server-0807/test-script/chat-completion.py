import asyncio
import aiohttp
import time
import json


async def send_streaming_request(session, url, payload, request_id, auth_token=None):
    try:
        start_time = time.time()
        first_token_time = None
        full_response = ""
        chunk_count = 0

        print(f"\n[Request {request_id}] Starting...")

        # Prepare headers
        headers = {"Content-Type": "application/json"}
        if auth_token:
            headers["Authorization"] = f"Bearer {auth_token}"

        async with session.post(url, json=payload, headers=headers) as response:
            if response.status != 200:
                print(f"[Request {request_id}] ERROR: HTTP {response.status}")
                return {
                    "request_id": request_id,
                    "status_code": response.status,
                    "success": False,
                    "error": f"HTTP {response.status}",
                    "response_time": time.time() - start_time,
                }

            print(f"[Request {request_id}] Connected, waiting for tokens...")

            # Process streaming response
            async for line in response.content:
                if line:
                    line_str = line.decode("utf-8").strip()

                    # Skip empty lines and non-data lines
                    if not line_str or not line_str.startswith("data: "):
                        continue

                    # Remove 'data: ' prefix
                    json_str = line_str[6:]

                    # Check for end of stream
                    if json_str == "[DONE]":
                        print(f"\n[Request {request_id}] ✓ COMPLETE")
                        break

                    try:
                        chunk = json.loads(json_str)
                        chunk_count += 1

                        # Debug: Show the first few chunks to understand structure
                        if chunk_count <= 2:
                            print(
                                f"\n[Request {request_id}] DEBUG - Chunk {chunk_count}: {json.dumps(chunk, indent=2)}"
                            )

                        # Record time to first token
                        if first_token_time is None:
                            first_token_time = time.time()
                            ttft = first_token_time - start_time
                            print(f"[Request {request_id}] First token in {ttft:.2f}s:")

                        # Extract text from chunk - try multiple possible paths
                        new_text = ""

                        # Method 1: OpenAI-style chat completions (delta.content)
                        if "choices" in chunk and len(chunk["choices"]) > 0:
                            choice = chunk["choices"][0]
                            if "delta" in choice and "content" in choice["delta"]:
                                new_text = choice["delta"]["content"]
                            # Method 2: OpenAI-style completions (delta.text)
                            elif "delta" in choice and "text" in choice["delta"]:
                                new_text = choice["delta"]["text"]
                            # Method 3: Direct text field
                            elif "text" in choice:
                                new_text = choice["text"]

                        # Method 4: Direct text field in root
                        elif "text" in chunk:
                            new_text = chunk["text"]

                        # Method 5: Other possible structures
                        elif "content" in chunk:
                            new_text = chunk["content"]

                        # Print the new text immediately as it arrives
                        if new_text:
                            print(f"[{request_id}]{new_text}", end="", flush=True)
                            full_response += new_text
                        elif (
                            chunk_count <= 3
                        ):  # Debug first few chunks if no text found
                            print(
                                f"\n[Request {request_id}] DEBUG - No text found in chunk {chunk_count}"
                            )
                            print(f"Available keys: {list(chunk.keys())}")

                    except json.JSONDecodeError:
                        continue

            end_time = time.time()

            return {
                "request_id": request_id,
                "status_code": response.status,
                "response_time": end_time - start_time,
                "time_to_first_token": (first_token_time - start_time)
                if first_token_time
                else None,
                "generated_text": full_response,
                "chunk_count": chunk_count,
                "success": True,
            }

    except Exception as e:
        end_time = time.time()
        print(f"\n[Request {request_id}] ERROR: {str(e)}")
        return {
            "request_id": request_id,
            "status_code": None,
            "response_time": end_time - start_time,
            "error": str(e),
            "success": False,
        }


async def load_test_streaming(num_requests=20, auth_token=None):
    url = "http://10.200.108.149:8000/v1/chat/completions"  # or /v1/chat/completions

    # Payload for streaming
    payload = {
        "model": "Qwen/Qwen3-32B-AWQ",
        "max_tokens": 100,
        "messages": [
            {
                "role": "user",
                "content": "Write a short story about a robot discovering emotions. Be creative and detailed.",
            }
        ],
        "temperature": 0.7,
        "stream": True,
    }

    print(f"🚀 Starting STREAMING load test with {num_requests} concurrent requests...")
    print(f"📡 Target URL: {url}")
    print(f"🔐 Auth token: {'✓ Provided' if auth_token else '✗ None'}")
    print(f"📝 Prompt: {payload['messages']}")
    print(f"🎯 Max tokens: {payload['max_tokens']}")
    print("=" * 80)
    print(
        "💡 TIP: Each request is prefixed with [Request ID] so you can follow individual streams"
    )
    print("=" * 80)

    async with aiohttp.ClientSession() as session:
        # Create tasks
        tasks = [
            send_streaming_request(session, url, payload, i + 1, auth_token)
            for i in range(num_requests)
        ]

        start_time = time.time()
        results = await asyncio.gather(*tasks)
        end_time = time.time()

    # Process results
    successful_requests = [r for r in results if r["success"]]
    failed_requests = [r for r in results if not r["success"]]

    print(f"\n\n{'=' * 80}")
    print("📊 STREAMING LOAD TEST RESULTS")
    print(f"{'=' * 80}")
    print(f"Total requests: {len(results)}")
    print(f"✅ Successful: {len(successful_requests)}")
    print(f"❌ Failed: {len(failed_requests)}")
    print(f"⏱️  Total time: {end_time - start_time:.2f} seconds")

    if successful_requests:
        # Calculate metrics
        response_times = [r["response_time"] for r in successful_requests]
        ttft_times = [
            r["time_to_first_token"]
            for r in successful_requests
            if r["time_to_first_token"]
        ]

        avg_response_time = sum(response_times) / len(response_times)
        min_response_time = min(response_times)
        max_response_time = max(response_times)

        print("\n📈 PERFORMANCE METRICS:")
        print(f"   Average total response time: {avg_response_time:.2f}s")
        print(f"   Min response time: {min_response_time:.2f}s")
        print(f"   Max response time: {max_response_time:.2f}s")
        print(
            f"   Requests per second: {len(successful_requests) / (end_time - start_time):.2f}"
        )

        if ttft_times:
            avg_ttft = sum(ttft_times) / len(ttft_times)
            min_ttft = min(ttft_times)
            max_ttft = max(ttft_times)
            print("\n⚡ TIME TO FIRST TOKEN:")
            print(f"   Average TTFT: {avg_ttft:.2f}s")
            print(f"   Min TTFT: {min_ttft:.2f}s")
            print(f"   Max TTFT: {max_ttft:.2f}s")

        total_chunks = sum(r["chunk_count"] for r in successful_requests)
        total_tokens = sum(len(r["generated_text"]) for r in successful_requests)
        print("\n🔄 STREAMING STATS:")
        print(f"   Total chunks received: {total_chunks}")
        print(
            f"   Average chunks per request: {total_chunks / len(successful_requests):.1f}"
        )
        print(f"   Total characters generated: {total_tokens}")
        print(
            f"   Average chars per request: {total_tokens / len(successful_requests):.0f}"
        )

    # Show failed requests
    if failed_requests:
        print("\n❌ FAILED REQUESTS:")
        for result in failed_requests:
            print(
                f"   Request {result['request_id']}: {result.get('error', 'Unknown error')}"
            )

    print(f"\n{'=' * 80}")
    print(
        "🎉 Test completed! You experienced real-time streaming from all concurrent requests."
    )
    print("💡 Notice how you could see responses arriving as they were generated!")

    return successful_requests, failed_requests


if __name__ == "__main__":
    print("🤖 vLLM Streaming Load Test")

    # Full load test
    asyncio.run(load_test_streaming(num_requests=20, auth_token=""))
