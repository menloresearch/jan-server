#!/bin/bash

curl -X POST 'http://10.200.108.149:1234/v1/chat/completions' \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer $1" \
-d '{
    "model": "jan-hq/Qwen3-14B-v0.2-deepresearch-no-think-100-step",
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
