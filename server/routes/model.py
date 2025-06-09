import requests
from fastapi import APIRouter, Request
from fastapi.exceptions import HTTPException
from fastapi.responses import StreamingResponse
from skills.deep_research.core import deep_research

from .limiter import limiter
from protocol.fastchat_openai import (
    ChatCompletionRequest,
)

from config import config

router = APIRouter(prefix="/v1", tags=["v1"])


####################################
# Deep Research route implementation
####################################


@router.post(
    "/deep/chat/completions",
    summary="Create chat completion",
    description="Generate a chat completion using the specified model and messages",
    response_description="Chat completion response or stream",
)
@limiter.limit("10/minute")
async def deep_chat_completions(
    request: Request,
    chat_request: ChatCompletionRequest,
    # api_key: str = Depends(validate_api_key),
):
    """Create a deep research completion"""

    messages = chat_request.messages
    content = ""

    for msg in messages:
        if msg["role"] == "user":
            content = msg["content"]

    return StreamingResponse(
        deep_research(content),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "Connection": "keep-alive"},
    )


@router.get(
    "/deep/models",
    summary="Show available models",
    description="Show a list of all the available models serve by the endpoint",
    response_description="JSON list of available model",
)
async def deep_models(
    request: Request,
    # api_key: str = Depends(validate_api_key),
):
    """Create a chat completion"""
    return proxy_to_vllm(
        "/models",
        "GET",
    )


######################################
# Standard OpenAI route implementation
######################################
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
    # api_key: str = Depends(validate_api_key),
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
    # api_key: str = Depends(validate_api_key),
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
    target_url = f"{config.model_base_url.rstrip('/')}/{endpoint.lstrip('/')}"

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
