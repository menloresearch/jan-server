import ast
import os

from openai import OpenAI
from routes.openai_protocol import (
    ChatCompletionResponse,
)
from .utils import SerperClient

from .prompt import (
    get_current_date,
    query_writer_instructions,
    web_searcher_instructions,
)
from .schema import (
    ChatCompletionUserMessage,
    GenerateQueryData,
)


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
        number_queries=3,
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
        print("QUERY: ", query, "\n")
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
