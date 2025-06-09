from pydantic import BaseModel
import os
from dotenv import load_dotenv

load_dotenv()


class Config(BaseModel):
    model_base_url: str = os.getenv("MODEL_BASE_URL", "")
    model_api_key: str = os.getenv("MODEL_API_KEY", "")
    search_api_key: str = os.getenv("SEARCH_API_KEY", "")
    port: int = int(os.getenv("PORT", "8000"))


config = Config()
