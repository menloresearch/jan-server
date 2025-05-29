import asyncio
import aiohttp
import time
import json


async def send_request(session, url, payload, request_id):
    try:
        start_time = time.time()
        async with session.post(url, json=payload) as response:
            result = await response.json()
            end_time = time.time()

            return {
                "request_id": request_id,
                "status_code": response.status,
                "response_time": end_time - start_time,
                "result": result,
                "success": True,
            }
    except Exception as e:
        end_time = time.time()
        return {
            "request_id": request_id,
            "status_code": None,
            "response_time": end_time - start_time,
            "result": None,
            "error": str(e),
            "success": False,
        }


async def load_test():
    url = "http://10.200.108.149:5000/v1/completions"
    payload = {
        "model": "Qwen/Qwen3-32B-AWQ",
        "prompt": "Tell me a story about",
        "max_tokens": 1000,
        "temperature": 0.7,
    }

    print("Starting load test with 20 concurrent requests...")
    print(f"Target URL: {url}")
    print("-" * 60)

    async with aiohttp.ClientSession() as session:
        # Create tasks with request IDs
        tasks = [send_request(session, url, payload, i + 1) for i in range(1)]

        start_time = time.time()
        results = await asyncio.gather(*tasks)
        end_time = time.time()

    # Process and display results
    successful_requests = [r for r in results if r["success"]]
    failed_requests = [r for r in results if not r["success"]]

    print("\n=== LOAD TEST RESULTS ===")
    print(f"Total requests: {len(results)}")
    print(f"Successful: {len(successful_requests)}")
    print(f"Failed: {len(failed_requests)}")
    print(f"Total time: {end_time - start_time:.2f} seconds")

    if successful_requests:
        avg_response_time = sum(r["response_time"] for r in successful_requests) / len(
            successful_requests
        )
        min_response_time = min(r["response_time"] for r in successful_requests)
        max_response_time = max(r["response_time"] for r in successful_requests)

        print(f"Average response time: {avg_response_time:.2f}s")
        print(f"Min response time: {min_response_time:.2f}s")
        print(f"Max response time: {max_response_time:.2f}s")
        print(
            f"Requests per second: {len(successful_requests) / (end_time - start_time):.2f}"
        )

    print("\n=== INDIVIDUAL RESPONSES ===")
    for result in results:
        if result["success"]:
            print(f"\nRequest {result['request_id']}: SUCCESS")
            print(f"  Response time: {result['response_time']:.2f}s")
            print(f"  Status code: {result['status_code']}")

            # Extract and display the generated text
            if "choices" in result["result"] and len(result["result"]["choices"]) > 0:
                generated_text = result["result"]["choices"][0].get("text", "").strip()
                print(f"  Generated text length: {len(generated_text)}")
            else:
                print(f"  Full response: {json.dumps(result['result'], indent=2)}")
        else:
            print(f"\nRequest {result['request_id']}: FAILED")
            print(f"  Error: {result['error']}")
            print(f"  Response time: {result['response_time']:.2f}s")

    # Show failed requests summary
    if failed_requests:
        print("\n=== FAILED REQUESTS SUMMARY ===")
        for result in failed_requests:
            print(f"Request {result['request_id']}: {result['error']}")


if __name__ == "__main__":
    # Update these values for your setup
    print("Remember to update the URL and model name in the script!")
    asyncio.run(load_test())
