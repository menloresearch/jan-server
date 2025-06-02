import requests
from fastapi import APIRouter, Depends, Request
from fastapi.exceptions import HTTPException
from fastapi.responses import StreamingResponse
from .openai_protocol import ChatCompletionRequest
from .api import validate_api_key
from .limiter import limiter
import os
from dotenv import load_dotenv

load_dotenv()

VLLM_SERVER_URL = os.getenv("VLLM_SERVER_URL")

router = APIRouter(prefix="/v1", tags=["v1"])


@router.post(
    "/chat/completions",
    summary="Create chat completion",
    description="Generate a chat completion using the specified model and messages",
    response_description="Chat completion response or stream",
)
@limiter.limit("10/minute")
async def chat_completions(
    request: Request,
    chat_request: ChatCompletionRequest,
    api_key: str = Depends(validate_api_key),
):
    """Create a chat completion"""
    return proxy_to_vllm(
        "/chat/completions",
        "POST",
        chat_request.model_dump(mode="json"),
    )


@router.get(
    "/models",
    summary="Show available models",
    description="Show a list of all the available models serve by the endpoint",
    response_description="JSON list of available model",
)
async def models(
    request: Request,
    api_key: str = Depends(validate_api_key),
):
    """Create a chat completion"""
    return proxy_to_vllm(
        "/models",
        "GET",
    )


def proxy_to_vllm(
    endpoint: str, method: str = "POST", json_data: dict = None, params: dict = None
):
    """Helper function to proxy requests to vLLM"""
    target_url = f"{VLLM_SERVER_URL.rstrip('/')}/v1/{endpoint.lstrip('/')}"

    try:
        response = requests.request(
            method=method,
            url=target_url,
            json=json_data,
            params=params,
            stream=True,
            timeout=300,
        )

        return StreamingResponse(
            response.iter_content(chunk_size=8192),
            status_code=response.status_code,
            headers={
                k: v
                for k, v in response.headers.items()
                if k.lower() != "content-encoding"
            },
        )

    except requests.RequestException as e:
        raise HTTPException(status_code=502, detail=f"vLLM server error: {str(e)}")
