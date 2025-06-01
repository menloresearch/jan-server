from typing import List, Optional

import requests
from fastapi import APIRouter, Depends, Request
from fastapi.exceptions import HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from typing import Literal, Dict, Any, Field, Union

from .api import validate_api_key
from .limiter import limiter

VLLM_SERVER_URL = "http://localhost:5000"

router = APIRouter(prefix="/v1", tags=["v1"])


class Function(BaseModel):
    name: str
    description: Optional[str] = None
    parameters: Optional[Dict[str, Any]] = None


class Tool(BaseModel):
    type: Literal["function"] = "function"
    function: Function


class ToolChoice(BaseModel):
    type: Literal["function"]
    function: Dict[str, str]  # {"name": "function_name"}


class ResponseFormat(BaseModel):
    type: Literal["text", "json_object"] = "text"


class ChatMessage(BaseModel):
    role: Literal["system", "user", "assistant", "tool"]
    content: Optional[str] = None
    name: Optional[str] = None
    tool_calls: Optional[List[Dict[str, Any]]] = None
    tool_call_id: Optional[str] = None


class ChatCompletionRequest(BaseModel):
    # Required
    model: str
    messages: List[ChatMessage]

    # Core generation parameters
    temperature: Optional[float] = Field(default=1.0, ge=0.0, le=2.0)
    max_tokens: Optional[int] = Field(default=None, gt=0)
    top_p: Optional[float] = Field(default=1.0, ge=0.0, le=1.0)

    # Streaming and stopping
    stream: Optional[bool] = False
    stop: Optional[Union[str, List[str]]] = None

    # Penalty parameters
    frequency_penalty: Optional[float] = Field(default=0.0, ge=-2.0, le=2.0)
    presence_penalty: Optional[float] = Field(default=0.0, ge=-2.0, le=2.0)

    # Tool calling
    tools: Optional[List[Tool]] = None
    tool_choice: Optional[Union[Literal["none", "auto"], ToolChoice]] = "auto"

    # Response format
    response_format: Optional[ResponseFormat] = None

    # Additional parameters
    n: Optional[int] = Field(default=1, ge=1, le=128)
    seed: Optional[int] = None
    logprobs: Optional[bool] = None
    top_logprobs: Optional[int] = Field(default=None, ge=0, le=20)

    # User identification
    user: Optional[str] = None

    # Legacy parameters (less commonly used)
    logit_bias: Optional[Dict[str, float]] = None

    # Model configuration options for function calling
    parallel_tool_calls: Optional[bool] = True


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
    return proxy_to_vllm("/chat/completions", "POST", chat_request.dict())


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
