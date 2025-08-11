from typing import List, Dict, Any, Optional
from pydantic import BaseModel


class ToolCall(BaseModel):
    id: str
    type: str = "function"
    function: Dict[str, Any]


class ChatMessage(BaseModel):
    role: str
    content: Optional[str] = None
    tool_calls: Optional[List[ToolCall]] = None
    tool_call_id: Optional[str] = None


class ResearchRequest(BaseModel):
    messages: List[Dict[str, Any]]
    model: str = "default"
    stream: bool = True
    tools: Optional[List[Dict[str, Any]]] = None


class MCPTool(BaseModel):
    name: str
    description: str
    inputSchema: Dict[str, Any]