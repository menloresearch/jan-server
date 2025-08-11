from typing import List
from pydantic import BaseModel


class GenerateQueryData(BaseModel):
    rationale: str
    query: List[str]


class ReflectionData(BaseModel):
    is_sufficient: str
    knowledge_gap: str
    follow_up_queries: List[str]


class ChatCompletionUserMessage(BaseModel):
    content: str
    """The contents of the user message."""

    role: str
    """The role of the messages author, in this case `user`."""
