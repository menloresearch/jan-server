import json
import os
import ast
import httpx

from config import config
from openai import AsyncOpenAI
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
    GenerateQueryData,
)
from .utils import (
    SerperClient,
    get_current_date,
    create_sse_message,
    create_message,
)


async def deep_research(request: str):
    """Generator function that yields streaming updates"""
    try:
        llm = AsyncOpenAI(
            api_key=config.model_api_key,
            base_url=config.model_base_url,
            timeout=httpx.Timeout(
                connect=30.0,  # Connection timeout
                read=120.0,  # Read timeout
                write=30.0,  # Write timeout
                pool=30.0,  # Pool timeout
            ),
            max_retries=3,
        )

        # Step 1: Generate query
        yield create_sse_message("[NOTIFY] Starting query generation...\n\n")

        query = await generate_query(llm, request)
        print("Query done")

        yield create_sse_message("[NOTIFY] Finished query generation\n\n")

        # # Step 2: Web research
        yield create_sse_message("[NOTIFY] Starting web research...\n\n")

        search_summary = await web_research(llm, request, query.query)
        print("Search done")

        yield create_sse_message("[NOTIFY] Finished web research\n\n")

        # Step 3: Reflection
        yield create_sse_message("[NOTIFY] Starting reflection...\n\n")

        reflection_result = await reflection(llm, request, search_summary)
        print("Reflection done")

        yield create_sse_message("[NOTIFY] Finished reflection\n\n")

        loop = 0

        while not reflection_result["is_sufficient"] and loop < config.max_search_loop:
            yield create_sse_message("[NOTIFY] Finding more infomration...\n\n")

            search_summary = await web_research(
                llm,
                request,
                reflection_result["follow_up_queries"],
            )

            yield create_sse_message("[NOTIFY] Starting reflection...\n\n")

            reflection_result = await reflection(llm, request, search_summary)

            yield create_sse_message("[NOTIFY] Finished reflection\n\n")

            loop += 1

        # Step 4: Finalize
        async for chunk in finalize_answer(llm, request, search_summary):
            yield create_sse_message(chunk)

    except Exception as e:
        yield create_sse_message(f"[ERROR] An error occurred: {str(e)}")
        print(f"Error: {str(e)}")

    print("Completed")
    yield b"data: [DONE]\n\n"


async def generate_query(llm, request):
    current_date = get_current_date()
    formatted_request = query_writer_instructions.format(
        current_date=current_date,
        research_topic=request,
        number_queries=1,
    )

    response = await llm.chat.completions.create(
        model=config.model_name,
        messages=[create_message("user", formatted_request)],
    )

    parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    content = parsed_response.choices[0].message.content

    parsed_content = GenerateQueryData.model_validate(ast.literal_eval(content.strip()))

    return parsed_content


async def web_research(llm, request, queries):
    serper = SerperClient(
        api_key=os.getenv("SERPER_API_KEY"),
    )

    formatted_prompt = web_searcher_instructions.format(
        current_date=get_current_date(),
        research_topic=request,
    )

    search_context = ""

    for query in queries:
        search_results = serper.search(query, num=3)
        search_context += serper.format_search_results(search_results) + "\n"

    response = await llm.chat.completions.create(
        model=config.model_name,
        messages=[
            create_message(
                "assistant",
                formatted_prompt,
            ),
            create_message(
                "user",
                f"Based on the following search results, provide a comprehensive analysis:\n\n{search_context}",
            ),
        ],
    )

    parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    content = parsed_response.choices[0].message.content

    return content


async def reflection(llm, request, summary):
    current_date = get_current_date()
    formatted_prompt = reflection_instructions.format(
        current_date=current_date,
        research_topic=request,
        summaries="\n\n---\n\n".join(summary),
    )

    response = await llm.chat.completions.create(
        model=config.model_name,
        messages=[create_message("assistant", formatted_prompt)],
    )

    parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    content = parsed_response.choices[0].message.content

    literal = json.loads(content.strip())
    # parsed_content = ReflectionData.model_validate_json(literal)

    return literal


async def finalize_answer(llm, request, summary):
    current_date = get_current_date()
    formatted_prompt = answer_instructions.format(
        current_date=current_date,
        research_topic=request,
        summaries="\n---\n\n".join(summary),
    )

    response = await llm.chat.completions.create(
        model=config.model_name,
        messages=[create_message("assistant", formatted_prompt)],
        stream=True,
    )

    async for chunk in response:
        if chunk.choices[0].delta.content is not None:
            yield chunk.choices[0].delta.content
