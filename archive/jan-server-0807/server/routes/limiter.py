from slowapi import Limiter


# Create a custom key function that uses API key instead of IP
def get_api_key_identifier(request):
    """Extract Bearer token from request headers for rate limiting"""
    auth_header = request.headers.get("Authorization")
    if auth_header and auth_header.startswith("Bearer "):
        api_key = auth_header.split(" ")[1]
        return f"api_key:{api_key}"


# Initialize limiter with custom key function
limiter = Limiter(
    key_func=get_api_key_identifier,
    storage_uri="memory://",
)
