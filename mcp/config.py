from pydantic import BaseModel
import os
from dotenv import load_dotenv

load_dotenv()


class Config(BaseModel):
    serper_api_key: str = os.getenv("SERPER_API_KEY", "")
    port: int = int(os.getenv("PORT", "8000"))


config = Config()
