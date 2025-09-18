# Conversation Flow Guide for Frontend Integration

This guide explains how to integrate with the Jan API Gateway's conversation system, including the completion API and response API flows.

## Two Ways to Use Conversations

### Method 1: Completion API with Conversation Management

The completion API automatically handles conversation creation and message appending.

#### Storage Options

The completion API supports two storage flags to control how messages are persisted:

- **`store`** (boolean, optional, default: `false`): When set to `true`, saves both the user message and assistant response to the conversation. When `false`, messages are not stored in the conversation history.

- **`store_reasoning`** (boolean, optional, default: `false`): When set to `true`, includes reasoning content in stored messages. This only takes effect when `store` is also `true`. Useful for models that provide reasoning explanations.

#### Flow:
1. **First Request** (No conversation ID):
   ```json
   POST /v1/chat/completions
   {
     "model": "jan-v1-4b",
     "messages": [
       {"role": "user", "content": "Hello, how are you?"}
     ],
     "stream": false,
     "store": true,
     "store_reasoning": false
   }
   ```

2. **Completion Response** (New conversation created):
   ```json
   {
     "id": "msg_oc07tomng5fqqi8w6bbxzmbuco19v3f9bq7xriuvpq",
     "object": "chat.completion",
     "created": 1234567890,
     "model": "jan-v1-4b",
     "choices": [
       {
         "message": {
           "role": "assistant",
           "content": "I'm doing well, thank you!"
         },
         "finish_reason": "stop"
       }
     ],
     "metadata": {
       "conversation_id": "conv_8zrnfsrj9d8424ngl0n2jbien0af3845gfhvpqc5un",
       "conversation_created": true,
       "conversation_title": "Hello, how are you?",
       "ask_item_id": "msg_049gu35s5kwj65tegn398fnut9o1o7p194xu6a61u3",
       "completion_item_id": "msg_oc07tomng5fqqi8w6bbxzmbuco19v3f9bq7xriuvpq"
     }
   }
   ```

3. **Continue Conversation** (Use conversation ID):
   ```json
   POST /v1/chat/completions
   {
     "model": "jan-v1-4b",
     "messages": [
       {"role": "user", "content": "What's the weather like?"}
     ],
     "conversation": "conv_uzaxr1z1mq38k23r99kl1qq9eelobeam0gw21n8q9z",
     "stream": false,
     "store": true,
     "store_reasoning": false
   }
   ```

#### Streaming Support:
For streaming responses, set `"stream": true` and handle Server-Sent Events (SSE):

```json
POST /v1/chat/completions
{
  "model": "jan-v1-4b",
  "messages": [
    {"role": "user", "content": "Hello, how are you?"}
  ],
  "stream": true,
  "store": true,
  "store_reasoning": false
}
```

**Streaming Response Format:**

The server sends multiple SSE events. The first event contains conversation metadata, followed by content chunks:

1. **Metadata Event** (sent first):
```
data: {"completion_item_id":"msg_oc07tomng5fqqi8w6bbxzmbuco19v3f9bq7xriuvpq","conversation_created":true,"conversation_id":"conv_8zrnfsrj9d8424ngl0n2jbien0af3845gfhvpqc5un","conversation_title":"333 Tell me name of largest ocean","object":"chat.completion.metadata","ask_item_id":"msg_049gu35s5kwj65tegn398fnut9o1o7p194xu6a61u3"}

```

**Metadata Event Attribute Meanings:**

- `conversation_created`: (boolean) Indicates if a new conversation was created as a result of this request.
- `conversation_id`: (string) The unique string identifier for the conversation. Use this for subsequent messages in the same conversation.
- `conversation_title`: (string) The title or summary of the conversation, often generated from the initial user message.
- `ask_item_id`: (string) The unique string identifier for the user's message that was just sent. This ID can be used to reference the specific ask message in the database.
- `completion_item_id`: (string) The unique string identifier for the assistant's response message. This ID can be used to reference the specific completion message in the database.
- `object`: (string) The type of object returned. For this event, it is always `"chat.completion.metadata"` to indicate metadata about the chat completion.

**Finish Reason Values:**

The `finish_reason` field indicates why the completion ended:

- `stop`: The model completed its response naturally
- `function_call`: The model is requesting to call a function (legacy format)
- `tool_calls`: The model is requesting to call one or more tools (new format)

2. **Content Chunk Events** (sent continuously):
```
data: {"id":"chatcmpl-b61389e4-eddf-935d-9ef4-7c9ab6a6d689","object":"chat.completion.chunk","created":1758067863,"model":"jan-v1-4b","choices":[{"index":0,"delta":{"role":"assistant","content":"I'm"},"logprobs":null,"finish_reason":null}]}

data: {"id":"chatcmpl-b61389e4-eddf-935d-9ef4-7c9ab6a6d689","object":"chat.completion.chunk","created":1758067863,"model":"jan-v1-4b","choices":[{"index":0,"delta":{"content":" doing"},"logprobs":null,"finish_reason":null}]}

data: {"id":"chatcmpl-b61389e4-eddf-935d-9ef4-7c9ab6a6d689","object":"chat.completion.chunk","created":1758067863,"model":"jan-v1-4b","choices":[{"index":0,"delta":{"content":" well, thank you!"},"logprobs":null,"finish_reason":"stop"}]}
```

**Example with Tool Calls:**
```
data: {"id":"chatcmpl-b61389e4-eddf-935d-9ef4-7c9ab6a6d689","object":"chat.completion.chunk","created":1758067863,"model":"jan-v1-4b","choices":[{"index":0,"delta":{"tool_calls":[{"id":"call_123","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"New York\"}"}}]},"logprobs":null,"finish_reason":"tool_calls"}]}
```

**Example with Function Call (Legacy):**
```
data: {"id":"chatcmpl-b61389e4-eddf-935d-9ef4-7c9ab6a6d689","object":"chat.completion.chunk","created":1758067863,"model":"jan-v1-4b","choices":[{"index":0,"delta":{"function_call":{"name":"get_weather","arguments":"{\"location\":\"New York\"}"}},"logprobs":null,"finish_reason":"function_call"}]}
```


2. **Continue Conversation** (Use conversation ID):
   ```json
   POST /v1/chat/completions
   {
     "model": "jan-v1-4b",
     "messages": [
       {"role": "user", "content": "What's the weather like?"}
     ],
     "conversation": "conv_uzaxr1z1mq38k23r99kl1qq9eelobeam0gw21n8q9z",
     "stream": false,
     "store": true,
     "store_reasoning": false
   }
   ```

### Method 2: Direct Conversation Management

Use the conversation API to explicitly manage conversations and their messages.

#### 1. Create Conversation:
```json
POST /v1/conversations
{
  "title": "My Chat Session",
  "metadata": {
    "model": "jan-v1-4b",
    "session_type": "chat"
  }
}
```

**Response:**
```json
{
  "id": "conv_abc123...",
  "object": "conversation",
  "title": "My Chat Session",
  "created_at": 1234567890,
  "metadata": {
    "model": "jan-v1-4b",
    "session_type": "chat"
  }
}
```

#### 2. Add Messages to Conversation:
```json
POST /v1/conversations/{conversation_id}/items
{
  "items": [
    {
      "type": "message",
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Hello, how are you?"
        }
      ]
    }
  ]
}
```

#### 3. List Conversation Items:
```http
GET /v1/conversations/{conversation_id}/items
```

#### 4. Delete Specific Conversation Items:
```http
GET /v1/conversations/{conversation_id}/items/{item_id}
```

#### 5. Update Conversation:
```json
PATCH /v1/conversations/{conversation_id}
{
  "title": "Updated Chat Title"
}
```

#### 6. List All Conversations:
```http
GET /v1/conversations?limit=20&after=cursor_id
```

### Common Error Codes

- `0199506b-314d-70e2-a8aa-d5fde1569d1d` - User not found
- `a1b2c3d4-e5f6-7890-abcd-ef1234567890` - Conversation not found
- `cf237451-8932-48d1-9cf6-42c4db2d4805` - Invalid request payload
- `c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc` - Unauthorized access

### HTTP Status Codes

- `200` - Success
- `400` - Bad Request (invalid payload)
- `401` - Unauthorized (invalid API key)
- `404` - Not Found (conversation/resource not found)
- `422` - Validation Error
- `429` - Rate Limit Exceeded
- `500` - Internal Server Error