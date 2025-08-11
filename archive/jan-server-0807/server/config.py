from pydantic import BaseModel
import os
from dotenv import load_dotenv

load_dotenv()


class Config(BaseModel):
    model_name: str = os.getenv("MODEL_NAME", "")
    model_base_url: str = os.getenv("MODEL_BASE_URL", "")
    model_api_key: str = os.getenv("MODEL_API_KEY", "")
    search_api_key: str = os.getenv("SEARCH_API_KEY", "")
    serper_api_key: str = os.getenv("SERPER_API_KEY", "")
    port: int = int(os.getenv("PORT", "8000"))
    max_search_loop: int = 3
    search_query_results: int = 5
    num_query_generated: int = 3


config = Config()
