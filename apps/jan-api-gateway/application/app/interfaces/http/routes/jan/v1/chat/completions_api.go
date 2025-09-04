package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// ExtendedChatCompletionRequest extends OpenAI's request with additional fields
type ExtendedChatCompletionRequest struct {
	openai.ChatCompletionRequest
	ParentMessageID string `json:"parent_message_id,omitempty"`
}

type CompletionAPI struct {
	userService         *user.UserService
	apikeyService       *apikey.ApiKeyService
	conversationService *conversation.ConversationService
}

func NewCompletionAPI(userService *user.UserService, apikeyService *apikey.ApiKeyService, conversationService *conversation.ConversationService) *CompletionAPI {
	return &CompletionAPI{
		userService:         userService,
		apikeyService:       apikeyService,
		conversationService: conversationService,
	}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", middleware.OptionalAuthMiddleware(), completionAPI.PostCompletion)
}

// ChatCompletionResponseSwagger is a doc-only version without http.Header
type ChatCompletionResponseSwagger struct {
	ID      string                        `json:"id"`
	Object  string                        `json:"object"`
	Created int64                         `json:"created"`
	Model   string                        `json:"model"`
	Choices []openai.ChatCompletionChoice `json:"choices"`
	Usage   openai.Usage                  `json:"usage"`
}

// CreateChatCompletion
// @Summary Create a chat completion
// @Description Generates a model response for the given chat conversation.
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ExtendedChatCompletionRequest true "ExtendedChatCompletionRequest payload"
// @Success 200 {object} ChatCompletionResponseSwagger "Successful response"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/chat/completions [post]
func (api *CompletionAPI) PostCompletion(reqCtx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
	key := "AnonymousUserKey"
	var currentUser *user.User

	if err != nil {
		// Log the error for debugging=
		fmt.Printf("DEBUG: Failed to get user claim: %v\n", err)
	} else if userClaim != nil {
		fmt.Printf("DEBUG: User claim found: %+v\n", userClaim)
		user, err := api.userService.FindByEmail(reqCtx, userClaim.Email)
		if err != nil {
			reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "62a772b9-58ec-4332-b669-920c7f4a8821",
				Error: fmt.Sprintf("User lookup failed: %v", err.Error()),
			})
			return
		}
		if user == nil {
			reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "62a772b9-58ec-4332-b669-920c7f4a8821",
				Error: fmt.Sprintf("User not found for email: %s", userClaim.Email),
			})
			return
		}
		currentUser = user
		fmt.Printf("DEBUG: Current user found: %+v\n", currentUser)
		key, err = api.getOrCreateUserKey(reqCtx, user)
		if err != nil {
			reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "7e29d138-8b8e-4895-8edc-c0876ebb1a52",
				Error: err.Error(),
			})
			return
		}
	} else {
		fmt.Printf("DEBUG: No user claim found\n")
	}

	// Parse as extended OpenAI format
	var extendedRequest ExtendedChatCompletionRequest
	if err := reqCtx.ShouldBindJSON(&extendedRequest); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "cf237451-8932-48d1-9cf6-42c4db2d4805",
			Error: "Invalid request format. Expected ExtendedChatCompletionRequest format.",
		})
		return
	}

	// Set default values for web interface compatibility
	if !extendedRequest.Stream {
		extendedRequest.Stream = true
	}
	if extendedRequest.Temperature == 0 {
		extendedRequest.Temperature = 0.7
	}
	if extendedRequest.MaxTokens == 0 {
		extendedRequest.MaxTokens = 1000
	}

	// Validate model exists
	modelRegistry := inferencemodelregistry.GetInstance()
	mToE := modelRegistry.GetModelToEndpoints()
	endpoints, ok := mToE[extendedRequest.Model]
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "59253517-df33-44bf-9333-c927402e4e2e",
			Error: fmt.Sprintf("Model: %s does not exist", extendedRequest.Model),
		})
		return
	}

	// Validate client exists
	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	clientExists := false
	for _, endpoint := range endpoints {
		if endpoint == janInferenceClient.BaseURL {
			clientExists = true
			break
		}
	}
	if !clientExists {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "6c6e4ea0-53d2-4c6c-8617-3a645af59f43",
			Error: "Client does not exist",
		})
		return
	}

	api.handleStreamingCompletion(reqCtx, key, extendedRequest.ChatCompletionRequest, currentUser, extendedRequest.ParentMessageID)
}

// handleStreamingCompletion handles streaming chat completions with conversation management
func (api *CompletionAPI) handleStreamingCompletion(reqCtx *gin.Context, key string, request openai.ChatCompletionRequest, currentUser *user.User, parentMessageID string) {
	// Set SSE headers
	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Handle conversation based on parent_message_id
	var conv *conversation.Conversation
	var userID uint
	if currentUser != nil {
		userID = currentUser.ID
		fmt.Printf("DEBUG: Looking for conversation with parent_message_id: %s, userID: %d\n", parentMessageID, userID)

		// Check if this is a new conversation or continuation
		if parentMessageID == "client-created-root" {
			// Create new conversation for first message
			conv, _ = api.conversationService.CreateConversation(reqCtx, userID, nil, true, map[string]string{
				"model": request.Model,
			})
			fmt.Printf("DEBUG: Created new conversation: %+v\n", conv)
		} else {
			// Find existing conversation by parent_message_id
			var err error
			conv, err = api.conversationService.GetConversationByPublicIDAndUserID(reqCtx, parentMessageID, userID)
			if err != nil {
				reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
					Code:  "8f7a2b1c-9d3e-4f5a-8b2c-1e4f5a8b2c3d",
					Error: fmt.Sprintf("Error finding conversation: %s", err.Error()),
				})
				return
			}
			if conv == nil {
				// If conversation not found, throw error
				reqCtx.JSON(http.StatusNotFound, responses.ErrorResponse{
					Code:  "9e8b7a6c-5d4e-3f2a-1b0c-9e8b7a6c5d4e",
					Error: fmt.Sprintf("Conversation with ID '%s' not found", parentMessageID),
				})
				return
			}
			fmt.Printf("DEBUG: Found existing conversation: %+v\n", conv)
		}
	} else {
		fmt.Printf("DEBUG: No current user, cannot handle conversation\n")
	}

	// Create wait groups for parallel processing
	var wg sync.WaitGroup
	var titleGenerated bool
	var titleMutex sync.Mutex

	// Generate conversation ID for response
	conversationID := conv.PublicID

	// Add user message to conversation
	userMessageID := "user-" + fmt.Sprintf("%d", time.Now().Unix())
	userContent := []conversation.Content{
		{
			Type: "text",
			Text: &conversation.Text{
				Value: request.Messages[len(request.Messages)-1].Content,
			},
		},
	}
	userRole := conversation.ItemRoleUser
	api.conversationService.AddItem(reqCtx, conv, userID, conversation.ItemTypeMessage, &userRole, userContent)

	// Send user message delta
	userDelta := map[string]interface{}{
		"v": map[string]interface{}{
			"message": map[string]interface{}{
				"id":          userMessageID,
				"author":      map[string]interface{}{"role": "user"},
				"create_time": float64(time.Now().Unix()),
				"content":     map[string]interface{}{"content_type": "text", "parts": []string{request.Messages[len(request.Messages)-1].Content}},
				"status":      "finished_successfully",
				"end_turn":    nil,
				"weight":      1.0,
				"metadata":    map[string]interface{}{},
				"recipient":   "all",
				"channel":     nil,
			},
			"conversation_id": conversationID,
			"error":           nil,
		},
		"c": 1,
	}
	api.sendSSEEvent(reqCtx, "delta", userDelta)

	// Start title generation in background (for all responses)
	wg.Add(1)
	go func() {
		defer wg.Done()
		api.generateTitle(reqCtx, request.Messages[len(request.Messages)-1].Content, conversationID, &titleGenerated, &titleMutex)
	}()

	// Stream the completion
	var fullResponse string
	var functionCall *openai.FunctionCall
	streamErr := api.streamCompletion(reqCtx, key, request, func(chunk string) {
		fullResponse += chunk
		// Send delta events for each chunk
		delta := map[string]interface{}{
			"p": "/message/content/parts/0",
			"o": "append",
			"v": chunk,
		}
		api.sendSSEEvent(reqCtx, "delta", delta)
	}, func(fc *openai.FunctionCall) {
		functionCall = fc
		// Send function call delta
		functionCallDelta := map[string]interface{}{
			"p": "/message/function_call",
			"o": "replace",
			"v": map[string]interface{}{
				"name":      fc.Name,
				"arguments": fc.Arguments,
			},
		}
		api.sendSSEEvent(reqCtx, "delta", functionCallDelta)
	})

	if streamErr != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "c3af973c-eada-4e8b-96d9-e92546588cd3",
			Error: streamErr.Error(),
		})
		return
	}

	// Wait for title generation to complete
	wg.Wait()

	// Send completion status
	completionDelta := map[string]interface{}{
		"p": "",
		"o": "patch",
		"v": []map[string]interface{}{
			{"p": "/message/status", "o": "replace", "v": "finished_successfully"},
			{"p": "/message/end_turn", "o": "replace", "v": true},
			{"p": "/message/metadata", "o": "append", "v": map[string]interface{}{
				"finish_details": map[string]interface{}{
					"type":        "stop",
					"stop_tokens": []int{200002},
				},
				"is_complete": true,
			}},
		},
	}
	api.sendSSEEvent(reqCtx, "delta", completionDelta)

	// Send stream complete
	streamComplete := map[string]interface{}{
		"type":            "message_stream_complete",
		"conversation_id": conversationID,
	}
	api.sendSSEData(reqCtx, streamComplete)

	// Send conversation detail metadata
	conversationMetadata := map[string]interface{}{
		"type":               "conversation_detail_metadata",
		"banner_info":        nil,
		"model_limits":       []interface{}{},
		"default_model_slug": request.Model,
		"conversation_id":    conversationID,
	}
	api.sendSSEData(reqCtx, conversationMetadata)

	// Send [DONE]
	api.sendSSEData(reqCtx, "[DONE]")

	// Save assistant message to conversation
	assistantContent := []conversation.Content{
		{
			Type: "text",
			Text: &conversation.Text{
				Value: fullResponse,
			},
		},
	}

	assistantRole := conversation.ItemRoleAssistant
	api.conversationService.AddItem(reqCtx, conv, userID, conversation.ItemTypeMessage, &assistantRole, assistantContent)

	// Add function call as a separate item if present
	if functionCall != nil {
		functionCallContent := []conversation.Content{
			{
				Type: "text",
				Text: &conversation.Text{
					Value: fmt.Sprintf("Function: %s\nArguments: %s", functionCall.Name, functionCall.Arguments),
				},
			},
		}

		// Create a separate function call item
		api.conversationService.AddItem(reqCtx, conv, userID, conversation.ItemTypeFunction, &assistantRole, functionCallContent)
	}

	// Update conversation title if generated
	titleMutex.Lock()
	if titleGenerated {
		// Generate title from user message
		title := api.generateTitleFromMessage(request.Messages[len(request.Messages)-1].Content)
		// Update conversation title
		api.conversationService.UpdateAndAuthorizeConversation(reqCtx, conv.PublicID, userID, ptr.ToString(title), nil)
	}
	titleMutex.Unlock()
}

// generateTitle generates a title for the conversation in the background
func (api *CompletionAPI) generateTitle(reqCtx *gin.Context, userMessage string, conversationID string, titleGenerated *bool, titleMutex *sync.Mutex) {
	// Generate title using simple logic
	title := api.generateTitleFromMessage(userMessage)

	titleMutex.Lock()
	*titleGenerated = true
	titleMutex.Unlock()

	// Send title generation event
	titleEvent := map[string]interface{}{
		"type":            "title_generation",
		"title":           title,
		"conversation_id": conversationID,
	}
	api.sendSSEData(reqCtx, titleEvent)
}

// generateTitleFromMessage generates a title from the user message
func (api *CompletionAPI) generateTitleFromMessage(userMessage string) string {
	// Simple title generation based on first few words
	if len(userMessage) > 50 {
		return userMessage[:50] + "..."
	}
	return userMessage
}

// streamCompletion streams the completion from the inference client
func (api *CompletionAPI) streamCompletion(reqCtx *gin.Context, key string, request openai.ChatCompletionRequest, onChunk func(string), onFunctionCall func(*openai.FunctionCall)) error {
	// Get chunks from the inference client
	chunkChan, err := janinference.NewJanInferenceClient(reqCtx).CreateChatCompletionStreamChunks(reqCtx, key, request)
	if err != nil {
		return err
	}

	// Track accumulated function call data
	var accumulatedFunctionCall *openai.FunctionCall
	var functionCallComplete bool

	// Process chunks
	for chunk := range chunkChan {
		// Skip empty chunks and [DONE]
		if len(chunk) == 0 || chunk == "data: [DONE]" {
			continue
		}

		// Extract content and function call from OpenAI streaming format
		content, functionCall := api.extractContentFromOpenAIStream(chunk)

		// Handle content
		if content != "" {
			onChunk(content)
		}

		// Handle function call if present
		if functionCall != nil && !functionCallComplete {
			// Initialize or update accumulated function call
			if accumulatedFunctionCall == nil {
				accumulatedFunctionCall = &openai.FunctionCall{
					Name:      functionCall.Name,
					Arguments: functionCall.Arguments,
				}
			} else {
				// Accumulate arguments if name is empty (streaming arguments)
				if functionCall.Name == "" && functionCall.Arguments != "" {
					accumulatedFunctionCall.Arguments += functionCall.Arguments
				} else if functionCall.Name != "" {
					// Update name if provided
					accumulatedFunctionCall.Name = functionCall.Name
					if functionCall.Arguments != "" {
						accumulatedFunctionCall.Arguments = functionCall.Arguments
					}
				}
			}

			// Check if function call is complete (has both name and complete arguments)
			if accumulatedFunctionCall.Name != "" && accumulatedFunctionCall.Arguments != "" {
				// Check if arguments are complete JSON
				if strings.HasSuffix(accumulatedFunctionCall.Arguments, "}") {
					functionCallComplete = true
					if onFunctionCall != nil {
						onFunctionCall(accumulatedFunctionCall)
					}
				}
			}
		}
	}
	return nil
}

// extractContentFromOpenAIStream extracts content from OpenAI streaming format
func (api *CompletionAPI) extractContentFromOpenAIStream(chunk string) (string, *openai.FunctionCall) {
	// Handle different streaming formats

	// Format 1: data: {"choices":[{"delta":{"content":"chunk"}}]}
	if len(chunk) >= 6 && chunk[:6] == "data: " {
		jsonStr := chunk[6:]

		// Parse the JSON with both function_call and tool_calls support
		var data struct {
			Choices []struct {
				Delta struct {
					Content          string               `json:"content"`
					ReasoningContent string               `json:"reasoning_content"`
					FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
					ToolCalls        []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Index    int    `json:"index"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls,omitempty"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &data); err == nil && len(data.Choices) > 0 {
			// Use reasoning_content if content is empty (jan-v1-4b model format)
			content := data.Choices[0].Delta.Content
			if content == "" {
				content = data.Choices[0].Delta.ReasoningContent
			}

			functionCall := data.Choices[0].Delta.FunctionCall

			// Handle tool_calls format
			if functionCall == nil && len(data.Choices[0].Delta.ToolCalls) > 0 {
				toolCall := data.Choices[0].Delta.ToolCalls[0]

				// Create function call even if name or arguments are empty (they will be accumulated)
				functionCall = &openai.FunctionCall{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				}
			}

			return content, functionCall
		}
	}

	// Format 2: Direct JSON without "data: " prefix
	// Try to parse as direct JSON
	var data struct {
		Choices []struct {
			Delta struct {
				Content          string               `json:"content"`
				ReasoningContent string               `json:"reasoning_content"`
				FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Index    int    `json:"index"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls,omitempty"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(chunk), &data); err == nil && len(data.Choices) > 0 {
		// Use reasoning_content if content is empty (jan-v1-4b model format)
		content := data.Choices[0].Delta.Content
		if content == "" {
			content = data.Choices[0].Delta.ReasoningContent
		}

		functionCall := data.Choices[0].Delta.FunctionCall

		// Handle tool_calls format
		if functionCall == nil && len(data.Choices[0].Delta.ToolCalls) > 0 {
			toolCall := data.Choices[0].Delta.ToolCalls[0]

			// Create function call even if name or arguments are empty (they will be accumulated)
			functionCall = &openai.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			}
		}

		return content, functionCall
	}

	// Format 3: Simple content string (fallback)
	// If it's just a string, return it as content
	if len(chunk) > 0 && chunk[0] == '"' && chunk[len(chunk)-1] == '"' {
		var content string
		if err := json.Unmarshal([]byte(chunk), &content); err == nil {
			return content, nil
		}
	}

	return "", nil
}

// getOrCreateUserKey gets or creates an API key for the user
func (api *CompletionAPI) getOrCreateUserKey(reqCtx *gin.Context, user *user.User) (string, error) {
	apikeyEntities, err := api.apikeyService.Find(reqCtx, apikey.ApiKeyFilter{
		OwnerID:   &user.ID,
		OwnerType: ptr.ToString(string(apikey.OwnerTypeAdmin)),
	}, nil)
	if err != nil {
		return "", err
	}

	// TODO: Should we provide a default key to user?
	if len(apikeyEntities) == 0 {
		key, hash, err := api.apikeyService.GenerateKeyAndHash(reqCtx, apikey.OwnerTypeEphemeral)
		if err != nil {
			return "", err
		}

		// TODO: OwnerTypeEphemeral
		entity, err := api.apikeyService.CreateApiKey(reqCtx, &apikey.ApiKey{
			KeyHash:        hash,
			PlaintextHint:  fmt.Sprintf("sk-..%s", key[len(key)-3:]),
			Description:    "Default Key For User",
			Enabled:        true,
			OwnerType:      string(apikey.OwnerTypeEphemeral),
			OwnerID:        &user.ID,
			OrganizationID: nil,
			Permissions:    "{}",
		})
		if err != nil {
			return "", err
		}
		apikeyEntities = []*apikey.ApiKey{
			entity,
		}
	}
	return apikeyEntities[0].KeyHash, nil
}

// sendSSEEvent sends a Server-Sent Event
func (api *CompletionAPI) sendSSEEvent(reqCtx *gin.Context, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(reqCtx.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
	reqCtx.Writer.Flush()
}

// sendSSEData sends data without an event type
func (api *CompletionAPI) sendSSEData(reqCtx *gin.Context, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(reqCtx.Writer, "data: %s\n\n", string(jsonData))
	reqCtx.Writer.Flush()
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PostChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type PostChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}
