from typing import List, Optional

import requests
from fastapi import APIRouter, Depends
from fastapi.exceptions import HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel

from .api import validate_api_key

VLLM_SERVER_URL = "http://localhost:5000"

router = APIRouter(prefix="/v1", tags=["v1"])


class ChatMessage(BaseModel):
    role: str
    content: str


class ChatCompletionRequest(BaseModel):
    model: str
    messages: List[ChatMessage]
    temperature: Optional[float] = 0.7
    max_tokens: Optional[int] = None
    stream: Optional[bool] = False
    stop: Optional[List[str]] = None
    top_p: Optional[float] = None
    frequency_penalty: Optional[float] = None
    presence_penalty: Optional[float] = None


class CompletionRequest(BaseModel):
    model: str
    prompt: str
    temperature: Optional[float] = 0.7
    max_tokens: Optional[int] = 100
    stream: Optional[bool] = False
    stop: Optional[List[str]] = None
    top_p: Optional[float] = None


class ModelInfo(BaseModel):
    id: str
    object: str = "model"
    created: int
    owned_by: str


class ModelsResponse(BaseModel):
    object: str = "list"
    data: List[ModelInfo]


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


@router.post(
    "/chat/completions",
    summary="Create chat completion",
    description="Generate a chat completion using the specified model and messages",
    response_description="Chat completion response or stream",
)
async def chat_completions(
    request: ChatCompletionRequest,
    api_key: str = Depends(validate_api_key),
):
    """Create a chat completion"""
    return proxy_to_vllm("/chat/completions", "POST", request.dict())
