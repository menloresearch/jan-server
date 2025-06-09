import json
import xml.etree.ElementTree as ET
from datetime import datetime
from typing import Any, Dict, List

import requests
from protocol.fastchat_openai import (
    ChatCompletionResponseStreamChoice,
    ChatCompletionStreamResponse,
    DeltaMessage,
)
from .schema import ChatCompletionUserMessage


def create_message(role: str, content: str) -> ChatCompletionUserMessage:
    return ChatCompletionUserMessage(role=role, content=content).model_dump(mode="json")


def create_sse_message(content: str) -> str:
    """Create a Server-Sent Events formatted message"""
    stream = ChatCompletionStreamResponse(
        choices=[
            ChatCompletionResponseStreamChoice(
                index=0,
                delta=DeltaMessage(content=content),
            )
        ],
        model="test",
    ).model_dump()

    chunk = f"data: {json.dumps(stream)}\n\n"
    return chunk.encode("utf-8")


def get_current_date():
    return datetime.now().strftime("%B %d, %Y")


def parse_xml_tags(xml_string):
    """Parse XML string and extract text from specific tags"""
    try:
        root = ET.fromstring(f"<root>{xml_string}</root>")

        results = {}

        for element in root:
            tag_name = element.tag
            tag_text = element.text if element.text else ""
            results[tag_name] = tag_text.strip()

        return results

    except ET.ParseError as e:
        print(f"XML parsing error: {e}")
        return {}


class SerperClient:
    """
    Simple Python client for Serper API

    Get your API key at: https://serper.dev/api-key
    """

    def __init__(self, api_key: str):
        self.api_key = api_key
        self.base_url = "https://google.serper.dev"
        self.headers = {"X-API-KEY": api_key, "Content-Type": "application/json"}

    def search(self, query: str, **kwargs) -> Dict[str, Any]:
        """
        Perform a web search using Serper API

        Args:
            query (str): The search query
            **kwargs: Additional parameters like gl, hl, num, etc.

        Returns:
            dict: JSON response from Serper API
        """
        payload = {"q": query}
        payload.update(kwargs)

        response = requests.post(
            f"{self.base_url}/search", headers=self.headers, data=json.dumps(payload)
        )

        response.raise_for_status()  # Raise an exception for bad status codes
        return response.json()

    def format_search_results(self, search_results: Dict[str, Any]) -> str:
        """Format Serper search results into readable context for the LLM."""
        formatted_parts = []

        # Add knowledge graph if available
        if "knowledgeGraph" in search_results:
            kg = search_results["knowledgeGraph"]
            formatted_parts.append(
                f"Knowledge Graph:\n{kg.get('title', '')} - {kg.get('description', '')}\n"
            )

        # Add organic results
        for i, result in enumerate(search_results.get("organic", []), 1):
            formatted_parts.append(
                f"[{i}] {result.get('title', '')}\n"
                f"URL: {result.get('link', '')}\n"
                f"Summary: {result.get('snippet', '')}\n"
            )

        return "\n".join(formatted_parts)

    def images(self, query: str, **kwargs) -> Dict[str, Any]:
        """Search for images"""
        payload = {"q": query}
        payload.update(kwargs)

        response = requests.post(
            f"{self.base_url}/images", headers=self.headers, data=json.dumps(payload)
        )

        response.raise_for_status()
        return response.json()

    def news(self, query: str, **kwargs) -> Dict[str, Any]:
        """Search for news"""
        payload = {"q": query}
        payload.update(kwargs)

        response = requests.post(
            f"{self.base_url}/news", headers=self.headers, data=json.dumps(payload)
        )

        response.raise_for_status()
        return response.json()

    def places(self, query: str, **kwargs) -> Dict[str, Any]:
        """Search for places"""
        payload = {"q": query}
        payload.update(kwargs)

        response = requests.post(
            f"{self.base_url}/places", headers=self.headers, data=json.dumps(payload)
        )

        response.raise_for_status()
        return response.json()

    def videos(self, query: str, **kwargs) -> Dict[str, Any]:
        """Search for videos"""
        payload = {"q": query}
        payload.update(kwargs)

        response = requests.post(
            f"{self.base_url}/videos", headers=self.headers, data=json.dumps(payload)
        )

        response.raise_for_status()
        return response.json()

    def _parse_snippets(self, results: dict) -> List[str]:
        snippets = []

        if results.get("answerBox"):
            answer_box = results.get("answerBox", {})
            if answer_box.get("answer"):
                return [answer_box.get("answer")]
            elif answer_box.get("snippet"):
                return [answer_box.get("snippet").replace("\n", " ")]
            elif answer_box.get("snippetHighlighted"):
                return answer_box.get("snippetHighlighted")

        if results.get("knowledgeGraph"):
            kg = results.get("knowledgeGraph", {})
            title = kg.get("title")
            entity_type = kg.get("type")
            if entity_type:
                snippets.append(f"{title}: {entity_type}.")
            description = kg.get("description")
            if description:
                snippets.append(description)
            for attribute, value in kg.get("attributes", {}).items():
                snippets.append(f"{title} {attribute}: {value}.")

        for result in results[self.result_key_for_type[self.type]][: self.k]:
            if "snippet" in result:
                snippets.append(result["snippet"])
            for attribute, value in result.get("attributes", {}).items():
                snippets.append(f"{attribute}: {value}.")

        if len(snippets) == 0:
            return ["No good Google Search Result was found"]
        return snippets
