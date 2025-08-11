import json
import httpx
from typing import List, Dict, Any, AsyncGenerator
from openai import AsyncOpenAI

from config import config
from logger import logger

from .prompt import research_system_prompt
from .utils import get_current_date, get_mcp_tools, call_mcp_tool, format_tool_result
from skills.deep_research.utils import create_sse_message, create_message


async def research_chat(messages: List[Dict[str, Any]]) -> AsyncGenerator[bytes, None]:
    """Main research chat function that handles tool calls and streaming responses"""
    try:
        # Initialize OpenAI client
        llm = AsyncOpenAI(
            api_key=config.model_api_key,
            base_url=config.model_base_url,
            timeout=httpx.Timeout(
                connect=300.0,
                read=300.0,
                write=300.0,
                pool=30.0,
            ),
            max_retries=3,
        )

        # Get available MCP tools
        yield create_sse_message("[NOTIFY] Fetching available tools...\n\n")
        mcp_tools = await get_mcp_tools()
        logger.info(f"Found {len(mcp_tools)} MCP tools")

        # Prepare system message with current date
        current_date = get_current_date()
        system_message = create_message(
            "system", research_system_prompt.format(current_date=current_date)
        )

        # Build conversation history
        conversation = [system_message] + messages

        # Start the research loop
        max_iterations = 10  # Prevent infinite loops
        iteration = 0

        while iteration < max_iterations:
            iteration += 1
            logger.info(f"Research iteration {iteration}")

            # Make request to LLM with tools
            yield create_sse_message(
                f"[NOTIFY] Thinking... (iteration {iteration})\n\n"
            )

            response = await llm.chat.completions.create(
                model=config.model_name,
                messages=conversation,
                tools=mcp_tools if mcp_tools else None,
                tool_choice="auto" if mcp_tools else None,
                stream=True,
            )

            # Collect the complete response while streaming
            full_content = ""
            tool_calls = []
            finish_reason = None

            async for chunk in response:
                delta = chunk.choices[0].delta

                # Stream content to client
                if delta.content is not None:
                    full_content += delta.content
                    yield create_sse_message(delta.content)

                # Collect tool calls if present
                if delta.tool_calls:
                    # Handle tool calls - they might come in multiple chunks
                    for tool_call in delta.tool_calls:
                        if tool_call.index is not None:
                            # Extend tool_calls list if necessary
                            while len(tool_calls) <= tool_call.index:
                                tool_calls.append(
                                    {
                                        "id": None,
                                        "type": "function",
                                        "function": {"name": "", "arguments": ""},
                                    }
                                )

                            # Update the tool call at the correct index
                            if tool_call.id:
                                tool_calls[tool_call.index]["id"] = tool_call.id
                            if tool_call.function:
                                if tool_call.function.name:
                                    tool_calls[tool_call.index]["function"]["name"] += (
                                        tool_call.function.name
                                    )
                                if tool_call.function.arguments:
                                    tool_calls[tool_call.index]["function"][
                                        "arguments"
                                    ] += tool_call.function.arguments

                # Check if streaming is finished
                if chunk.choices[0].finish_reason:
                    finish_reason = chunk.choices[0].finish_reason

            # Now construct the complete assistant message
            assistant_message = {
                "role": "assistant",
                "content": full_content if full_content else None,
            }

            # Add tool_calls only if they exist and are complete
            if tool_calls:
                # Clean up any incomplete tool calls
                complete_tool_calls = [
                    tc for tc in tool_calls if tc["id"] and tc["function"]["name"]
                ]
                if complete_tool_calls:
                    assistant_message["tool_calls"] = complete_tool_calls

            # Add assistant message to conversation
            conversation.append(assistant_message)

            logger.debug(f"Conversation: \n\n{conversation}")

            # Check if there are tool calls to execute
            if assistant_message.get("tool_calls"):
                tool_calls = assistant_message["tool_calls"]
                yield create_sse_message(
                    f"[NOTIFY] Executing {len(tool_calls)} tool call(s)...\n\n"
                )

                # Execute each tool call
                for tool_call in tool_calls:
                    tool_name = tool_call["function"]["name"]
                    tool_args = json.loads(tool_call["function"]["arguments"])

                    logger.info(f"Calling tool: {tool_name} with args: {tool_args}")
                    yield create_sse_message(f"[NOTIFY] Calling {tool_name}...\n\n")

                    # Call the MCP tool
                    tool_result = await call_mcp_tool(tool_name, tool_args)
                    formatted_result = format_tool_result(tool_name, tool_result)
                    logger.info(
                        f"Result from calling tool: {tool_name} with args: {tool_args}\n\n{formatted_result}"
                    )

                    # Add tool result to conversation
                    conversation.append(
                        {
                            "role": "tool",
                            "content": formatted_result,
                            "tool_call_id": tool_call["id"],
                        }
                    )

                    yield create_sse_message(f"[NOTIFY] Completed {tool_name}\n\n")

                # Continue the loop to get the next response from LLM
                continue

            else:
                # # No tool calls, stream the final response
                # yield create_sse_message("[NOTIFY] Generating final response...\n\n")
                #
                # # Make a streaming request for the final response
                # stream_response = await llm.chat.completions.create(
                #     model=config.model_name,
                #     messages=conversation,
                #     stream=True,
                # )
                #
                # async for chunk in stream_response:
                #     if chunk.choices[0].delta.content is not None:
                #         yield create_sse_message(chunk.choices[0].delta.content)

                break  # Exit the loop after streaming final response

    except Exception as e:
        logger.error(f"Research error: {str(e)}")
        yield create_sse_message(f"[ERROR] An error occurred: {str(e)}")

    logger.info("Research workflow completed")
    yield b"data: [DONE]\n\n"


async def simple_research_response(user_message: str) -> AsyncGenerator[bytes, None]:
    """Simplified research function for single user messages"""
    messages = [create_message("user", user_message)]
    async for chunk in research_chat(messages):
        yield chunk
