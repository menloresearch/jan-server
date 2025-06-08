from pydantic import BaseModel


class KeyRequest(BaseModel):
    user: str


class KeyResponse(BaseModel):
    api_key: str


class KeyData(BaseModel):
    user: str
    created_at: str
