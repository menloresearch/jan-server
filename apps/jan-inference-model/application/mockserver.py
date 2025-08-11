from fastapi import FastAPI
from pydantic import BaseModel
from typing import List, Optional
import time
import uuid



class Message(BaseModel):
    role: str 
    content: str 

class ChatCompletionRequest(BaseModel):
    model: str 
    messages: List[Message] 
    temperature: Optional[float] = 0.7 
    max_tokens: Optional[int] = None 

class ChoiceMessage(BaseModel):
    role: str = "assistant"
    content: str = "This is a dummy response from the mock API."

class Choice(BaseModel):
    index: int = 0 
    message: ChoiceMessage 
    finish_reason: str = "stop" 

class ChatCompletionResponse(BaseModel):
    id: str
    object: str = "chat.completion"
    created: int
    model: str
    choices: List[Choice]

app = FastAPI()

@app.post("/v1/chat/completions", response_model=ChatCompletionResponse)
async def create_chat_completion(completion_request: ChatCompletionRequest):
    response_id = str(uuid.uuid4())
    
    dummy_response = ChatCompletionResponse(
        id=response_id,
        created=int(time.time()),
        model="jan-v1",
        choices=[Choice(
            index=0,
            message=ChoiceMessage(content="This is a dummy response based on your request."),
            finish_reason="stop"
        )]
    )

    return dummy_response

@app.get("/")
async def test():
    return {
        "service": "jan-inference-model"
    }