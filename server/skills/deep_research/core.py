import ast
import json
import os
import asyncio

from config import config
from openai import OpenAI
from protocol.fastchat_openai import (
    ChatCompletionResponse,
)

from .prompt import (
    query_writer_instructions,
    web_searcher_instructions,
    reflection_instructions,
    answer_instructions,
)
from .schema import (
    ChatCompletionUserMessage,
    GenerateQueryData,
)
from .utils import (
    SerperClient,
    get_current_date,
    create_sse_message,
)


async def deep_research(request: str):
    """Generator function that yields streaming updates"""
    try:
        llm = OpenAI(
            api_key=config.model_api_key,
            base_url=config.model_base_url,
        )

        # Step 1: Generate query
        yield create_sse_message("[NOTIFY] Starting query generation...")
        await asyncio.sleep(0)

        query = generate_query(llm, request)

        yield create_sse_message("[NOTIFY] Finished query generation")

        # Step 2: Web research
        yield create_sse_message("[NOTIFY] Starting web research...")
        await asyncio.sleep(0)

        search_summary = web_research(llm, request, query.query)

        yield create_sse_message("[NOTIFY] Finished web research")

        # Step 3: Reflection
        yield create_sse_message("[NOTIFY] Starting reflection...")
        await asyncio.sleep(0)

        reflection_result = reflection(llm, request, search_summary)
        print(reflection_result)
        yield create_sse_message("[NOTIFY] Finished reflection...")

        loop = 0

        while not reflection_result["is_sufficient"] and loop < config.max_search_loop:
            yield create_sse_message("[NOTIFY] Starting web research...")
            await asyncio.sleep(0)

            search_summary = web_research(
                llm,
                request,
                reflection_result["follow_up_queries"],
            )

            yield create_sse_message("[NOTIFY] Finished web research")

            yield create_sse_message("[NOTIFY] Starting reflection...")
            await asyncio.sleep(0)

            reflection_result = reflection(llm, request, search_summary)
            print(reflection_result)
            yield create_sse_message("[NOTIFY] Finished reflection...")

            loop += 1

        # Step 4: Finalize
        async for chunk in finalize_answer(llm, request, search_summary):
            yield create_sse_message(chunk)

    except Exception as e:
        yield create_sse_message(f"[ERROR] An error occurred: {str(e)}")
        print(f"Error: {str(e)}")

    print("Completed")
    yield b"data: [DONE]\n\n"


def generate_query(llm, request):
    # Format the prompt
    current_date = get_current_date()
    formatted_request = query_writer_instructions.format(
        current_date=current_date,
        research_topic=request,
        number_queries=1,
    )

    response = llm.chat.completions.create(
        model=config.model_name,
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
        model=config.model_name,
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
        model=config.model_name,
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
        model=config.model_name,
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
