import asyncio
import os
import json

import requests
from dotenv import load_dotenv
from fastapi import APIRouter, Request
from fastapi.exceptions import HTTPException
from fastapi.responses import StreamingResponse
from skills.deep_research.core import deep_research

from .limiter import limiter
from .openai_protocol import (
    ChatCompletionRequest,
    ChatCompletionResponseStreamChoice,
    ChatCompletionStreamResponse,
    DeltaMessage,
)

load_dotenv()

VLLM_SERVER_URL = os.getenv("VLLM_SERVER_URL")

router = APIRouter(prefix="/v1", tags=["v1"])


@router.get(
    "/test",
)
async def test():
    return await deep_research("Who is Yuuki from Menlo Research")


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
    # return proxy_to_vllm(
    #     "/chat/completions",
    #     "POST",
    #     chat_request.model_dump(mode="json"),
    # )

    async def generate_responses():
        response = await deep_research("Research who is Yuuki from Menlo Research")

        stream = ChatCompletionStreamResponse(
            id=response.id,
            choices=[
                ChatCompletionResponseStreamChoice(
                    index=0,
                    delta=DeltaMessage(content=response.choices[0].message.content),
                )
            ],
            model=response.model,
        )

        responses = [
            stream.model_dump(),
            stream.model_dump(),
            stream.model_dump(),
        ]

        for response in responses:
            # Format similar to OpenAI/vLLM streaming
            chunk = f"data: {json.dumps(response)}\n\n"
            yield chunk.encode("utf-8")  # Return bytes like iter_content
            await asyncio.sleep(1)

        # End stream marker (common in LLM APIs)
        yield b"data: [DONE]\n\n"

    return StreamingResponse(
        generate_responses(),
        media_type="text/plain",  # Must be this for SSE
        headers={"Cache-Control": "no-cache", "Connection": "keep-alive"},
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
