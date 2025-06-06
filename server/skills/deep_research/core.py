import ast
import os

from openai import OpenAI
from routes.openai_protocol import (
    ChatCompletionResponse,
)

from .prompt import (
    get_current_date,
    query_writer_instructions,
)
from .schema import (
    ChatCompletionUserMessage,
    GenerateQueryData,
)


async def deep_research(request):
    query = generate_query(request)
    print(query.query[0])

    return "test"


def generate_query(request):
    # Format the prompt
    current_date = get_current_date()
    formatted_request = query_writer_instructions.format(
        current_date=current_date,
        research_topic=request,
        number_queries=3,
    )

    llm = OpenAI(
        api_key=os.getenv("MENLO_API_KEY"),
        base_url="http://10.200.108.157:8080/v1",
        # base_url="http://10.200.108.149:5000/v1",
    )

    response = llm.chat.completions.create(
        model="Menlo_Deepresearch_Qwen3_14B",
        # model="Qwen/Qwen3-32B-AWQ",
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
