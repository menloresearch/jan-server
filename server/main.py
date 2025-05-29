from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import uvicorn

from routes import api, model


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

# def proxy_request(request: Request, path: str):
#     """Forward the request to vLLM server using requests"""
#
#     # Construct target URL
#     target_url = f"{VLLM_SERVER_URL.rstrip('/')}/{path.lstrip('/')}"
#
#     # Prepare headers (remove auth and host)
#     headers = {
#         k: v
#         for k, v in request.headers.items()
#         if k.lower() not in ["authorization", "host"]
#     }
#
#     try:
#         # Make the request to vLLM
#         response = requests.request(
#             method=request.method,
#             url=target_url,
#             headers=headers,
#             json=request.state.body if hasattr(request.state, "body") else None,
#             params=request.query_params,
#             stream=True,  # Support streaming responses
#         )
#
#         # Return streaming response to preserve SSE/streaming
#         return StreamingResponse(
#             response.iter_content(chunk_size=8192),
#             status_code=response.status_code,
#             headers=dict(response.headers),
#         )
#
#     except requests.RequestException as e:
#         raise HTTPException(
#             status_code=502,
#             detail={
#                 "error": {
#                     "message": f"Failed to connect to vLLM server: {str(e)}",
#                     "type": "api_error",
#                 }
#             },
#         )
#
#
# # Middleware to capture request body
# @app.middleware("http")
# async def capture_body(request: Request, call_next):
#     if request.method in ["POST", "PUT", "PATCH"]:
#         body = await request.body()
#         request.state.body = await request.json() if body else None
#     response = await call_next(request)
#     return response
#
#
# @app.api_route("/{path:path}", methods=["GET", "POST", "PUT", "DELETE", "PATCH"])
# async def proxy_all(
#     path: str,
#     request: Request,
#     # api_key: str = Depends(validate_api_key),
# ):
#     """Proxy all requests to vLLM server"""
#     return proxy_request(request, path)
#
#
# @app.get("/health")
# async def health_check():
#     """Health check endpoint"""
#     return {"status": "healthy"}


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
