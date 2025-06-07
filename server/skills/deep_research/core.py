import ast
import json
import os
import asyncio

from openai import OpenAI
from routes.openai_protocol import (
    ChatCompletionResponse,
    ChatCompletionResponseStreamChoice,
    ChatCompletionStreamResponse,
    DeltaMessage,
)

from .prompt import (
    get_current_date,
    query_writer_instructions,
    web_searcher_instructions,
    reflection_instructions,
    answer_instructions,
)
from .schema import (
    ChatCompletionUserMessage,
    GenerateQueryData,
)
from .utils import SerperClient


async def deep_research_stream(request: str):
    """Generator function that yields streaming updates"""
    try:
        llm = OpenAI(
            api_key=os.getenv("MENLO_API_KEY"),
            base_url="http://10.200.108.149:1234/v1",
        )

        # Step 1: Generate query
        yield create_sse_message("notify", "Starting query generation...")
        await asyncio.sleep(0)
        print("Starting query gen")

        query = generate_query(llm, request)
        yield create_sse_message("notify", "Finished query generation")
        yield create_sse_message("results", query.query)
        print("Finished query gen")

        # Step 2: Web research
        yield create_sse_message("notify", "Starting web research...")
        await asyncio.sleep(0)
        print("Starting web research")

        search_summary = web_research(llm, request, query.query)

        yield create_sse_message("notify", "Finished web research")
        print("Finished summary")

        yield create_sse_message("notify", "Starting reflection...")
        await asyncio.sleep(0)
        print("Starting reflection")

        reflection_result = reflection(llm, request, search_summary)
        print(reflection_result)
        yield create_sse_message("notify", "Finished reflection...")

        loop = 0

        while not reflection_result["is_sufficient"] and loop < 3:
            yield create_sse_message("notify", "Starting web research...")
            await asyncio.sleep(0)
            print("Starting web research")

            search_summary = web_research(
                llm,
                request,
                reflection_result["follow_up_queries"],
            )

            yield create_sse_message("notify", "Finished web research")
            print("Finished summary")

            yield create_sse_message("notify", "Starting reflection...")
            await asyncio.sleep(0)
            print("Starting reflection")

            reflection_result = reflection(llm, request, search_summary)
            print(reflection_result)
            yield create_sse_message("notify", "Finished reflection...")

            loop += 1

        async for chunk in finalize_answer(llm, request, search_summary):
            yield create_finalise_message(chunk)

    except Exception as e:
        yield create_sse_message("error", f"An error occurred: {str(e)}")
        print(f"Error: {str(e)}")

    print("Completed")
    yield b"data: [DONE]\n\n"


def create_finalise_message(content: str) -> str:
    """Create a Server-Sent Events formatted message"""
    stream = ChatCompletionStreamResponse(
        choices=[
            ChatCompletionResponseStreamChoice(
                index=0,
                delta=DeltaMessage(content=content),
            )
        ],
        model="test",
    ).model_dump()

    chunk = f"data: {json.dumps(stream)}\n\n"
    return chunk.encode("utf-8")


def create_sse_message(message_type: str, content: str) -> str:
    """Create a Server-Sent Events formatted message"""

    data = {"message": message_type, "content": content}

    stream = ChatCompletionStreamResponse(
        choices=[
            ChatCompletionResponseStreamChoice(
                index=0,
                delta=DeltaMessage(content=f"{json.dumps(data)}\n\n"),
            )
        ],
        model="test",
    ).model_dump()

    chunk = f"data: {json.dumps(stream)}\n\n"
    return chunk.encode("utf-8")


async def deep_research(request):
    llm = OpenAI(
        api_key=os.getenv("MENLO_API_KEY"),
        base_url="http://10.200.108.149:1234/v1",
    )

    query = generate_query(llm, request)
    print("[LOG] Finish generate_query")
    search_summary = web_research(llm, request, query.query)
    print("[LOG] Finish search_summary")

    return search_summary


def generate_query(llm, request):
    # Format the prompt
    current_date = get_current_date()
    formatted_request = query_writer_instructions.format(
        current_date=current_date,
        research_topic=request,
        number_queries=1,
    )

    response = llm.chat.completions.create(
        model="jan-hq/Qwen3-14B-v0.2-deepresearch-no-think-100-step",
        messages=[
            ChatCompletionUserMessage(
                role="user",
                content=formatted_request,
            ).model_dump(mode="json"),
        ],
    )

    parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    reasoning = parsed_response.choices[0].message.reasoning_content
    content = parsed_response.choices[0].message.content

    parsed_content = GenerateQueryData.model_validate(ast.literal_eval(content.strip()))

    return parsed_content


def web_research(llm, request, queries):
    serper = SerperClient(
        api_key=os.getenv("SERPER_API_KEY"),
    )

    formatted_prompt = web_searcher_instructions.format(
        current_date=get_current_date(),
        research_topic=request,
    )

    # Perform search and get structured results
    search_context = ""

    for query in queries:
        search_results = serper.search(query, num=3)
        search_context += serper.format_search_results(search_results) + "\n"

    # Generate analysis using OpenAI
    response = llm.chat.completions.create(
        model="jan-hq/Qwen3-14B-v0.2-deepresearch-no-think-100-step",
        messages=[
            ChatCompletionUserMessage(
                role="system",
                content=formatted_prompt,
            ).model_dump(mode="json"),
            ChatCompletionUserMessage(
                role="user",
                content=f"Based on the following search results, provide a comprehensive analysis:\n\n{search_context}",
            ).model_dump(mode="json"),
        ],
    )

    parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    reasoning = parsed_response.choices[0].message.reasoning_content
    content = parsed_response.choices[0].message.content

    # sources_gathered = extract_sources_from_results(search_results)
    # citations = create_citations(sources_gathered)
    # modified_text = insert_citation_markers(response.content, citations)

    # return {
    #     "sources_gathered": sources_gathered,
    #     "search_query": [state["search_query"]],
    #     "web_research_result": [modified_text],
    # }

    return content


def reflection(llm, request, summary):
    current_date = get_current_date()
    formatted_prompt = reflection_instructions.format(
        current_date=current_date,
        research_topic=request,
        summaries="\n\n---\n\n".join(summary),
    )
    # init Reasoning Model
    response = llm.chat.completions.create(
        model="jan-hq/Qwen3-14B-v0.2-deepresearch-no-think-100-step",
        messages=[
            ChatCompletionUserMessage(
                role="system",
                content=formatted_prompt,
            ).model_dump(mode="json"),
        ],
    )

    parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    reasoning = parsed_response.choices[0].message.reasoning_content
    content = parsed_response.choices[0].message.content

    literal = json.loads(content.strip())

    print(literal)

    # parsed_content = ReflectionData.model_validate_json(literal)

    return literal


async def finalize_answer(llm, request, summary):
    current_date = get_current_date()
    formatted_prompt = answer_instructions.format(
        current_date=current_date,
        research_topic=request,
        summaries="\n---\n\n".join(summary),
    )

    response = llm.chat.completions.create(
        model="jan-hq/Qwen3-14B-v0.2-deepresearch-no-think-100-step",
        messages=[
            ChatCompletionUserMessage(
                role="system",
                content=formatted_prompt,
            ).model_dump(mode="json"),
        ],
        stream=True,
    )

    # parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    # reasoning = parsed_response.choices[0].message.reasoning_content
    # content = parsed_response.choices[0].message.content

    for chunk in response:
        if chunk.choices[0].delta.content is not None:
            yield chunk.choices[0].delta.content
