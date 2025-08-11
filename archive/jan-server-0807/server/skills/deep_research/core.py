import os
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
    clean_and_parse_json,
    get_current_date,
    create_sse_message,
    create_message,
)

from logger import logger


async def deep_research(request: str):
    """Generator function that yields streaming updates"""
    try:
        llm = AsyncOpenAI(
            api_key=config.model_api_key,
            base_url=config.model_base_url,
            timeout=httpx.Timeout(
                connect=300.0,  # Connection timeout
                read=300.0,  # Read timeout
                write=300.0,  # Write timeout
                pool=30.0,
            ),
            max_retries=3,
        )

        # Step 1: Generate query
        yield create_sse_message("[NOTIFY] Starting query generation...\n\n")
        logger.info(f"Running {generate_query.__name__}")

        query = await generate_query(llm, request)
        logger.debug(f"{generate_query.__name__} results: \n\n{query}")

        yield create_sse_message("[NOTIFY] Finished query generation\n\n")
        logger.info(f"Complete {generate_query.__name__}")

        # # Step 2: Web research
        yield create_sse_message("[NOTIFY] Starting web research...\n\n")
        logger.info(f"Running {web_research.__name__}")

        search_summary = await web_research(llm, request, query.query)
        logger.debug(f"{web_research.__name__} results: \n\n{search_summary}")

        yield create_sse_message("[NOTIFY] Finished web research\n\n")
        logger.info(f"Complete {web_research.__name__}")

        # Step 3: Reflection
        yield create_sse_message("[NOTIFY] Starting reflection...\n\n")
        logger.info(f"Running {reflection.__name__}")

        reflection_result = await reflection(llm, request, search_summary)
        logger.debug(f"{reflection.__name__} results: \n\n{reflection_result}")

        yield create_sse_message("[NOTIFY] Finished reflection\n\n")
        logger.info(f"Complete {reflection.__name__}")

        loop = 0

        while not reflection_result["is_sufficient"] and loop < config.max_search_loop:
            yield create_sse_message("[NOTIFY] Finding more information...\n\n")
            logger.info(f"Running {web_research.__name__}")

            search_summary += "\n\n" + await web_research(
                llm,
                request,
                reflection_result["follow_up_queries"],
            )
            logger.debug(f"{web_research.__name__} results: \n\n{search_summary}")

            yield create_sse_message("[NOTIFY] Starting reflection...\n\n")
            logger.info(f"Running {reflection.__name__}")

            reflection_result = await reflection(llm, request, search_summary)
            logger.debug(f"{reflection.__name__} results: \n\n{reflection_result}")

            yield create_sse_message("[NOTIFY] Finished reflection\n\n")
            logger.info(f"Complete {reflection.__name__}")

            loop += 1

        # Step 4: Finalize
        logger.info(f"Running {finalize_answer.__name__}")
        async for chunk in finalize_answer(llm, request, search_summary):
            yield create_sse_message(chunk)

    except Exception as e:
        logger.error(f"{e}")
        yield create_sse_message(f"[ERROR] An error occurred: {str(e)}")
        print(f"Error: {str(e)}")

    logger.info("Complete Deep Research workflow")
    yield b"data: [DONE]\n\n"


async def generate_query(llm, request):
    current_date = get_current_date()
    formatted_request = query_writer_instructions.format(
        current_date=current_date,
        research_topic=request,
        number_queries=config.num_query_generated,
    )

    response = await llm.chat.completions.create(
        model=config.model_name,
        messages=[create_message("user", formatted_request)],
    )

    parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    content = parsed_response.choices[0].message.content
    content = clean_and_parse_json(content)

    parsed_content = GenerateQueryData.model_validate(content)

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
        search_results = serper.search(query, num=config.search_query_results)
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
        summaries=summary,
    )

    response = await llm.chat.completions.create(
        model=config.model_name,
        messages=[create_message("assistant", formatted_prompt)],
    )

    parsed_response = ChatCompletionResponse.model_validate(response.model_dump())
    content = parsed_response.choices[0].message.content
    content = clean_and_parse_json(content)

    # parsed_content = ReflectionData.model_validate(content)

    return content


async def finalize_answer(llm, request, summary):
    current_date = get_current_date()
    formatted_prompt = answer_instructions.format(
        current_date=current_date,
        research_topic=request,
        summaries=summary,
    )

    response = await llm.chat.completions.create(
        model=config.model_name,
        messages=[create_message("assistant", formatted_prompt)],
        stream=True,
    )

    async for chunk in response:
        if chunk.choices[0].delta.content is not None:
            yield chunk.choices[0].delta.content
