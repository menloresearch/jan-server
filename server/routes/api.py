import hashlib
import secrets
import uuid
from typing import Dict

from fastapi import APIRouter, Depends
from fastapi.exceptions import HTTPException
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from .schema import KeyRequest, KeyData, KeyResponse

router = APIRouter(prefix="/api", tags=["api"])

# TODO: Change to database later on
API_KEYS = {}


security = HTTPBearer()


def validate_api_key(credentials: HTTPAuthorizationCredentials = Depends(security)):
    """Validate the API key from the Authorization header"""
    api_key = credentials.credentials

    if api_key not in API_KEYS:
        raise HTTPException(
            status_code=401,
            detail={
                "error": {"message": "Invalid API key", "type": "invalid_request_error"}
            },
        )
    return api_key


@router.post("/gen", response_model=KeyResponse)
async def generate_api_key(request: KeyRequest):
    if not request.user or not request.user.strip():
        raise HTTPException(status_code=400, detail="User field is required")

    # Generate a secure random component
    random_bytes = secrets.token_bytes(16)
    timestamp = str(uuid.uuid4().hex)[:8]

    # Create hash from user + random data
    hash_input = f"{request.user}{random_bytes.hex()}{timestamp}"
    hash_value = hashlib.sha256(hash_input.encode()).hexdigest()[:16]

    # Format: sk-[hash]-[username]
    api_key = f"sk-{hash_value}-{request.user}"

    # Store the generated key
    from datetime import datetime

    API_KEYS[api_key] = {
        "user": request.user,
        "created_at": datetime.now().isoformat(),
    }

    return KeyResponse(api_key=api_key)


@router.get("/keys", response_model=Dict[str, KeyData])
async def get_all_api_keys():
    return API_KEYS
