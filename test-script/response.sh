#!/bin/bash

curl -X POST 'http://10.200.108.149:8000/v1/chat/completions' \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer $1" \
-d '{
    "model": "Qwen/Qwen3-32B-AWQ",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful math tutor. Guide the user through the solution step by step."
      },
      {
        "role": "user",
        "content": "how can I solve 8x + 7 = -23"
      }
    ]
}'
