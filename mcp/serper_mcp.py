import aiohttp
from typing import Dict, Any, Optional, Literal
from fastmcp import FastMCP
from config import config

serper_mcp = FastMCP("SerperMCP")


async def make_serper_request(endpoint: str, payload: Dict[str, Any]) -> Dict[str, Any]:
    """Make a request to Serper.dev API"""
    headers = {
        "X-API-KEY": config.serper_api_key,
        "Content-Type": "application/json",
    }

    async with aiohttp.ClientSession() as session:
        async with session.post(
            f"https://google.serper.dev/{endpoint}", headers=headers, json=payload
        ) as response:
            if response.status != 200:
                error_text = await response.text()
                raise Exception(f"Serper API error ({response.status}): {error_text}")

            return await response.json()


def build_search_query(
    q: str,
    site: Optional[str] = None,
    filetype: Optional[str] = None,
    inurl: Optional[str] = None,
    intitle: Optional[str] = None,
    related: Optional[str] = None,
    cache: Optional[str] = None,
    before: Optional[str] = None,
    after: Optional[str] = None,
    exact: Optional[str] = None,
    exclude: Optional[str] = None,
    or_terms: Optional[str] = None,
) -> str:
    """Build advanced search query with operators"""
    query_parts = []

    # Add exact phrase if specified
    if exact:
        query_parts.append(f'"{exact}"')

    # Add main query
    query_parts.append(q)

    # Add site restriction
    if site:
        query_parts.append(f"site:{site}")

    # Add filetype restriction
    if filetype:
        query_parts.append(f"filetype:{filetype}")

    # Add URL content search
    if inurl:
        query_parts.append(f"inurl:{inurl}")

    # Add title content search
    if intitle:
        query_parts.append(f"intitle:{intitle}")

    # Add related sites
    if related:
        query_parts.append(f"related:{related}")

    # Add cache search
    if cache:
        query_parts.append(f"cache:{cache}")

    # Add date restrictions
    if before:
        query_parts.append(f"before:{before}")

    if after:
        query_parts.append(f"after:{after}")

    # Add exclusions
    if exclude:
        exclude_terms = [term.strip() for term in exclude.split(",")]
        for term in exclude_terms:
            query_parts.append(f"-{term}")

    # Add OR terms
    if or_terms:
        or_list = [term.strip() for term in or_terms.split(",")]
        if len(or_list) > 1:
            or_query = " OR ".join(or_list)
            query_parts.append(f"({or_query})")

    return " ".join(query_parts)


@serper_mcp.tool()
async def google_search(
    q: str,
    gl: str = "us",
    hl: str = "en",
    location: Optional[str] = None,
    num: int = 10,
    tbs: Optional[Literal["qdr:h", "qdr:d", "qdr:w", "qdr:m", "qdr:y"]] = None,
    page: int = 1,
    autocorrect: bool = True,
    # Advanced search operators
    site: Optional[str] = None,
    filetype: Optional[str] = None,
    inurl: Optional[str] = None,
    intitle: Optional[str] = None,
    related: Optional[str] = None,
    cache: Optional[str] = None,
    before: Optional[str] = None,
    after: Optional[str] = None,
    exact: Optional[str] = None,
    exclude: Optional[str] = None,
    or_terms: Optional[str] = None,
) -> Dict[str, Any]:
    """
    Perform comprehensive Google search with advanced operators

    Args:
        q: Search query string (e.g., 'artificial intelligence', 'climate change solutions')
        gl: Region code for search results in ISO 3166-1 alpha-2 format (e.g., 'us', 'gb', 'de')
        hl: Language code for search results in ISO 639-1 format (e.g., 'en', 'es', 'fr')
        location: Location for search results (e.g., 'SoHo, New York, United States')
        num: Number of results to return (default: 10)
        tbs: Time-based search filter ('qdr:h', 'qdr:d', 'qdr:w', 'qdr:m', 'qdr:y')
        page: Page number of results to return (default: 1)
        autocorrect: Whether to autocorrect spelling in query
        site: Limit results to specific domain (e.g., 'github.com', 'wikipedia.org')
        filetype: Limit to specific file types (e.g., 'pdf', 'doc', 'xls')
        inurl: Search for pages with word in URL (e.g., 'download', 'tutorial')
        intitle: Search for pages with word in title (e.g., 'review', 'how to')
        related: Find similar websites (e.g., 'github.com', 'stackoverflow.com')
        cache: View Google's cached version of a specific URL (e.g., 'example.com/page')
        before: Date before in YYYY-MM-DD format (e.g., '2024-01-01')
        after: Date after in YYYY-MM-DD format (e.g., '2023-01-01')
        exact: Exact phrase match (e.g., 'machine learning', 'quantum computing')
        exclude: Terms to exclude from search results as comma-separated string
        or_terms: Alternative terms as comma-separated string

    Returns:
        Dictionary containing comprehensive search results
    """

    # Build advanced search query
    enhanced_query = build_search_query(
        q=q,
        site=site,
        filetype=filetype,
        inurl=inurl,
        intitle=intitle,
        related=related,
        cache=cache,
        before=before,
        after=after,
        exact=exact,
        exclude=exclude,
        or_terms=or_terms,
    )

    # Build payload
    payload = {
        "q": enhanced_query,
        "gl": gl,
        "hl": hl,
        "num": min(max(num, 1), 100),  # Clamp between 1-100
        "page": max(page, 1),
        "autocorrect": autocorrect,
    }

    # Add optional parameters
    if location:
        payload["location"] = location

    if tbs:
        payload["tbs"] = tbs

    try:
        result = await make_serper_request("search", payload)

        # Format comprehensive response
        formatted_result = {
            "searchParameters": {
                "query": q,
                "enhanced_query": enhanced_query,
                "gl": gl,
                "hl": hl,
                "location": location,
                "num": num,
                "page": page,
                "tbs": tbs,
                "autocorrect": autocorrect,
            },
            "searchInformation": {
                "totalResults": result.get("searchInformation", {}).get(
                    "totalResults", "0"
                ),
                "timeTaken": result.get("searchInformation", {}).get("timeTaken", 0),
                "formattedTotalResults": result.get("searchInformation", {}).get(
                    "formattedTotalResults", "0"
                ),
                "formattedTimeTaken": result.get("searchInformation", {}).get(
                    "formattedTimeTaken", "0 seconds"
                ),
            },
            "organic": [],
            "peopleAlsoAsk": result.get("peopleAlsoAsk", []),
            "relatedSearches": result.get("relatedSearches", []),
            "knowledgeGraph": result.get("knowledgeGraph"),
            "answerBox": result.get("answerBox"),
            "topStories": result.get("topStories", []),
            "videos": result.get("videos", []),
            "images": result.get("images", []),
            "shopping": result.get("shopping", []),
            "sitelinks": result.get("sitelinks", []),
        }

        # Process organic results with enhanced information
        for item in result.get("organic", []):
            organic_item = {
                "position": item.get("position", 0),
                "title": item.get("title", ""),
                "link": item.get("link", ""),
                "snippet": item.get("snippet", ""),
                "date": item.get("date"),
                "sitelinks": item.get("sitelinks", []),
                "richSnippet": item.get("richSnippet"),
                "rating": item.get("rating"),
                "ratingCount": item.get("ratingCount"),
                "priceRange": item.get("priceRange"),
            }
            formatted_result["organic"].append(organic_item)

        # Process People Also Ask with enhanced structure
        formatted_paa = []
        for paa in result.get("peopleAlsoAsk", []):
            formatted_paa.append(
                {
                    "question": paa.get("question", ""),
                    "snippet": paa.get("snippet", ""),
                    "title": paa.get("title", ""),
                    "link": paa.get("link", ""),
                }
            )
        formatted_result["peopleAlsoAsk"] = formatted_paa

        return formatted_result

    except Exception as e:
        return {
            "error": f"Search failed: {str(e)}",
            "searchParameters": {
                "query": q,
                "enhanced_query": enhanced_query,
                "gl": gl,
                "hl": hl,
            },
        }


@serper_mcp.tool()
async def read_page(
    url: str,
    include_markdown: bool = False,
    include_html: bool = False,
    selector: Optional[str] = None,
) -> Dict[str, Any]:
    """
    Read a webpage using provided url with comprehensive options

    Args:
        url: The URL to scrape
        include_markdown: Whether to include markdown content (default: False)
        include_html: Whether to include raw HTML in response (default: False)
        selector: CSS selector to extract specific elements (optional)

    Returns:
        Dictionary containing scraped content with metadata
    """
    payload = {"url": url}

    # Add optional parameters
    if include_html:
        payload["includeHtml"] = True

    if include_markdown:
        payload["includeMarkdown"] = True

    if selector:
        payload["selector"] = selector

    try:
        result = await make_serper_request("scrape", payload)

        # Format comprehensive response
        formatted_result = {
            "url": url,
            "title": result.get("title", ""),
            "text": result.get("text", ""),
            "status": result.get("status", "unknown"),
            "metadata": {
                "description": result.get("description", ""),
                "keywords": result.get("keywords", ""),
                "author": result.get("author", ""),
                "publishedDate": result.get("publishedDate", ""),
                "modifiedDate": result.get("modifiedDate", ""),
                "canonical": result.get("canonical", ""),
                "language": result.get("language", ""),
                "favicon": result.get("favicon", ""),
            },
            "headMetadata": result.get("headMetadata", {}),
            "jsonLd": result.get("jsonLd", []),
        }

        # Add optional fields if present and requested
        if include_html and "html" in result:
            formatted_result["html"] = result["html"]

        if include_markdown and "markdown" in result:
            formatted_result["markdown"] = result["markdown"]

        if "links" in result:
            formatted_result["links"] = result["links"]

        if "images" in result:
            formatted_result["images"] = result["images"]

        # Add additional metadata if available
        if "breadcrumbs" in result:
            formatted_result["breadcrumbs"] = result["breadcrumbs"]

        if "socialMedia" in result:
            formatted_result["socialMedia"] = result["socialMedia"]

        return formatted_result

    except Exception as e:
        return {
            "error": f"Scraping failed: {str(e)}",
            "url": url,
            "metadata": {
                "attempted_options": {
                    "include_markdown": include_markdown,
                    "include_html": include_html,
                    "selector": selector,
                }
            },
        }


# Create a separate function for the combined search and scrape functionality
# async def _search_and_read_internal(
#     query: str,
#     scrape_top_results: int = 3,
#     gl: str = "us",
#     hl: str = "en",
#     include_markdown: bool = False,
#     **search_kwargs,
# ) -> Dict[str, Any]:
#     """
#     Internal function to perform search and automatically scrape top results
#     """
#     try:
#         # Perform search by calling the google_search function directly
#         search_result = await google_search(q=query, gl=gl, hl=hl, **search_kwargs)
#
#         if "error" in search_result:
#             return search_result
#
#         # Scrape top results
#         scraped_results = []
#         organic_results = search_result.get("organic", [])
#
#         for i, result in enumerate(organic_results[:scrape_top_results]):
#             url = result.get("link", "")
#             if url:
#                 # Call the scrape function directly
#                 scraped_content = await read_page(
#                     url=url, include_markdown=include_markdown
#                 )
#                 scraped_results.append(
#                     {
#                         "position": result.get("position", i + 1),
#                         "title": result.get("title", ""),
#                         "url": url,
#                         "search_snippet": result.get("snippet", ""),
#                         "scraped_content": scraped_content,
#                     }
#                 )
#
#         return {
#             "searchResults": search_result,
#             "scrapedContent": scraped_results,
#             "summary": {
#                 "query": query,
#                 "total_search_results": len(organic_results),
#                 "scraped_results_count": len(scraped_results),
#                 "search_time": search_result.get("searchInformation", {}).get(
#                     "timeTaken", 0
#                 ),
#             },
#         }
#
#     except Exception as e:
#         return {"error": f"Search with scrape failed: {str(e)}", "query": query}


# @serper_mcp.tool()
# async def search_with_read(
#     query: str,
#     scrape_top_results: int = 3,
#     gl: str = "us",
#     hl: str = "en",
#     include_markdown: bool = False,
#     # Additional search parameters
#     location: Optional[str] = None,
#     num: int = 10,
#     tbs: Optional[str] = None,
#     page: int = 1,
#     autocorrect: bool = True,
#     site: Optional[str] = None,
#     filetype: Optional[str] = None,
#     inurl: Optional[str] = None,
#     intitle: Optional[str] = None,
#     related: Optional[str] = None,
#     cache: Optional[str] = None,
#     before: Optional[str] = None,
#     after: Optional[str] = None,
#     exact: Optional[str] = None,
#     exclude: Optional[str] = None,
#     or_terms: Optional[str] = None,
# ) -> Dict[str, Any]:
#     """
#     Perform search and automatically scrape top results for comprehensive information
#
#     Args:
#         query: Search query
#         scrape_top_results: Number of top results to scrape (default: 3)
#         gl: Region code (default: us)
#         hl: Language code (default: en)
#         include_markdown: Include markdown in scraped content
#         location: Location for search results
#         num: Number of search results to return
#         tbs: Time-based search filter
#         page: Page number of results
#         autocorrect: Whether to autocorrect spelling
#         site: Limit results to specific domain
#         filetype: Limit to specific file types
#         inurl: Search for pages with word in URL
#         intitle: Search for pages with word in title
#         related: Find similar websites
#         cache: View cached version of URL
#         before: Date before in YYYY-MM-DD format
#         after: Date after in YYYY-MM-DD format
#         exact: Exact phrase match
#         exclude: Terms to exclude (comma-separated)
#         or_terms: Alternative terms (comma-separated)
#
#     Returns:
#         Dictionary with search results and scraped content
#     """
#     # Prepare search kwargs
#     search_kwargs = {
#         "location": location,
#         "num": num,
#         "tbs": tbs,
#         "page": page,
#         "autocorrect": autocorrect,
#         "site": site,
#         "filetype": filetype,
#         "inurl": inurl,
#         "intitle": intitle,
#         "related": related,
#         "cache": cache,
#         "before": before,
#         "after": after,
#         "exact": exact,
#         "exclude": exclude,
#         "or_terms": or_terms,
#     }
#
#     # Filter out None values
#     search_kwargs = {k: v for k, v in search_kwargs.items() if v is not None}
#
#     return await _search_and_read_internal(
#         query=query,
#         scrape_top_results=scrape_top_results,
#         gl=gl,
#         hl=hl,
#         include_markdown=include_markdown,
#         **search_kwargs,
#     )


@serper_mcp.tool()
async def health_check() -> Dict[str, str]:
    """
    Check if the MCP server and Serper API are working

    Returns:
        Status information
    """
    try:
        # Test with a simple search
        _ = await make_serper_request("search", {"q": "test", "num": 1})
        return {
            "status": "healthy",
            "serper_api": "connected",
            "message": "MCP server is running and Serper API is accessible",
        }
    except Exception as e:
        return {
            "status": "unhealthy",
            "serper_api": "error",
            "message": f"Health check failed: {str(e)}",
        }
