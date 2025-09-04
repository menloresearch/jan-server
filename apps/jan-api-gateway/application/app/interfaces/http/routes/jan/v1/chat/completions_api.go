package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"slices"

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

// Constants for magic values
const (
	DefaultTemperature = 0.7
	DefaultMaxTokens   = 1000
	AnonymousUserKey   = ""
	ClientCreatedRoot  = "client-created-root"
	RequestTimeout     = 120 * time.Second
)

// ExtendedChatCompletionRequest extends OpenAI's request with additional fields
type ExtendedChatCompletionRequest struct {
	openai.ChatCompletionRequest
	ParentMessageID string `json:"parent_message_id,omitempty"`
}

// FunctionCallAccumulator handles streaming function call accumulation
type FunctionCallAccumulator struct {
	Name      string
	Arguments string
	Complete  bool
}

func (fca *FunctionCallAccumulator) AddChunk(functionCall *openai.FunctionCall) {
	if functionCall.Name != "" {
		fca.Name = functionCall.Name
	}
	if functionCall.Arguments != "" {
		fca.Arguments += functionCall.Arguments
	}

	// Check if complete
	if fca.Name != "" && fca.Arguments != "" && strings.HasSuffix(fca.Arguments, "}") {
		fca.Complete = true
	}
}

// SSEResponseBuilder handles building SSE responses
type SSEResponseBuilder struct {
	conversationID string
}

func (b *SSEResponseBuilder) SendUserMessage(reqCtx *gin.Context, messageID string, content string) {
	userDelta := map[string]interface{}{
		"v": map[string]interface{}{
			"message": map[string]interface{}{
				"id":          messageID,
				"author":      map[string]interface{}{"role": "user"},
				"create_time": float64(time.Now().Unix()),
				"content":     map[string]interface{}{"content_type": "text", "parts": []string{content}},
				"status":      "finished_successfully",
				"end_turn":    nil,
				"weight":      1.0,
				"metadata":    map[string]interface{}{},
				"recipient":   "all",
				"channel":     nil,
			},
			"conversation_id": b.conversationID,
			"error":           nil,
		},
		"c": 1,
	}
	sendSSEEvent(reqCtx, "delta", userDelta)
}

func (b *SSEResponseBuilder) SendContentChunk(reqCtx *gin.Context, chunk string) {
	delta := map[string]interface{}{
		"p": "/message/content/parts/0",
		"o": "append",
		"v": chunk,
	}
	sendSSEEvent(reqCtx, "delta", delta)
}

func (b *SSEResponseBuilder) SendFunctionCall(reqCtx *gin.Context, functionCall *openai.FunctionCall) {
	functionCallDelta := map[string]interface{}{
		"p": "/message/function_call",
		"o": "replace",
		"v": map[string]interface{}{
			"name":      functionCall.Name,
			"arguments": functionCall.Arguments,
		},
	}
	sendSSEEvent(reqCtx, "delta", functionCallDelta)
}

func (b *SSEResponseBuilder) SendCompletion(reqCtx *gin.Context) {
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
	sendSSEEvent(reqCtx, "delta", completionDelta)
}

func (b *SSEResponseBuilder) SendStreamComplete(reqCtx *gin.Context) {
	streamComplete := map[string]interface{}{
		"type":            "message_stream_complete",
		"conversation_id": b.conversationID,
	}
	sendSSEData(reqCtx, streamComplete)
}

func (b *SSEResponseBuilder) SendConversationMetadata(reqCtx *gin.Context, model string) {
	conversationMetadata := map[string]interface{}{
		"type":               "conversation_detail_metadata",
		"banner_info":        nil,
		"model_limits":       []interface{}{},
		"default_model_slug": model,
		"conversation_id":    b.conversationID,
	}
	sendSSEData(reqCtx, conversationMetadata)
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

// validateChatRequest validates the chat completion request
func (api *CompletionAPI) validateChatRequest(request *ExtendedChatCompletionRequest) error {
	if len(request.Messages) == 0 {
		return fmt.Errorf("messages cannot be empty")
	}

	if request.Model == "" {
		return fmt.Errorf("model is required")
	}

	return nil
}

// validateModelAndClient validates that the model exists and client is available
func (api *CompletionAPI) validateModelAndClient(model string) error {
	modelRegistry := inferencemodelregistry.GetInstance()
	mToE := modelRegistry.GetModelToEndpoints()
	endpoints, ok := mToE[model]
	if !ok {
		return fmt.Errorf("model: %s does not exist", model)
	}

	janInferenceClient := janinference.NewJanInferenceClient(context.Background())
	clientExists := slices.Contains(endpoints, janInferenceClient.BaseURL)
	if !clientExists {
		return fmt.Errorf("client does not exist")
	}

	return nil
}

// getOrCreateConversation handles conversation retrieval or creation
func (api *CompletionAPI) getOrCreateConversation(reqCtx *gin.Context, userID uint, parentMessageID string, model string) (*conversation.Conversation, error) {
	if parentMessageID == ClientCreatedRoot {
		return api.conversationService.CreateConversation(reqCtx, userID, nil, true, map[string]string{
			"model": model,
		})
	}

	conv, err := api.conversationService.GetConversationByPublicIDAndUserID(reqCtx, parentMessageID, userID)
	if err != nil {
		return nil, fmt.Errorf("error finding conversation: %w", err)
	}

	if conv == nil {
		return nil, fmt.Errorf("conversation with ID '%s' not found", parentMessageID)
	}

	return conv, nil
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
	// Add timeout context
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), RequestTimeout)
	defer cancel()

	// Use ctx for long-running operations
	userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
	key := AnonymousUserKey
	var currentUser *user.User

	if err != nil {
		// Log the error for debugging
		fmt.Printf("DEBUG: Failed to get user claim: %v\n", err)
	} else if userClaim != nil {
		fmt.Printf("DEBUG: User claim found: %+v\n", userClaim)
		user, err := api.userService.FindByEmail(ctx, userClaim.Email)
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

	// Validate request
	if err := api.validateChatRequest(&extendedRequest); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Error: err.Error(),
		})
		return
	}

	// Set default values for web interface compatibility
	if !extendedRequest.Stream {
		extendedRequest.Stream = true
	}
	if extendedRequest.Temperature == 0 {
		extendedRequest.Temperature = DefaultTemperature
	}
	if extendedRequest.MaxTokens == 0 {
		extendedRequest.MaxTokens = DefaultMaxTokens
	}

	// Validate model and client
	if err := api.validateModelAndClient(extendedRequest.Model); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "59253517-df33-44bf-9333-c927402e4e2e",
			Error: err.Error(),
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

		var err error
		conv, err = api.getOrCreateConversation(reqCtx, userID, parentMessageID, request.Model)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				reqCtx.JSON(http.StatusNotFound, responses.ErrorResponse{
					Code:  "9e8b7a6c-5d4e-3f2a-1b0c-9e8b7a6c5d4e",
					Error: err.Error(),
				})
			} else {
				reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
					Code:  "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
					Error: err.Error(),
				})
			}
			return
		}
		fmt.Printf("DEBUG: Conversation handled: %+v\n", conv)
	} else {
		fmt.Printf("DEBUG: No current user, cannot handle conversation\n")
	}

	// Create wait groups for parallel processing
	var wg sync.WaitGroup
	var titleGenerated bool
	var titleMutex sync.Mutex

	// Generate conversation ID for response
	conversationID := conv.PublicID

	// Initialize SSE response builder
	sseBuilder := &SSEResponseBuilder{conversationID: conversationID}

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
	sseBuilder.SendUserMessage(reqCtx, userMessageID, request.Messages[len(request.Messages)-1].Content)

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
		sseBuilder.SendContentChunk(reqCtx, chunk)
	}, func(fc *openai.FunctionCall) {
		functionCall = fc
		// Send function call delta
		sseBuilder.SendFunctionCall(reqCtx, fc)
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
	sseBuilder.SendCompletion(reqCtx)

	// Send stream complete
	sseBuilder.SendStreamComplete(reqCtx)

	// Send conversation detail metadata
	sseBuilder.SendConversationMetadata(reqCtx, request.Model)

	// Send [DONE]
	sendSSEData(reqCtx, "[DONE]")

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
	sendSSEData(reqCtx, titleEvent)
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

	// Use FunctionCallAccumulator for better function call handling
	accumulator := &FunctionCallAccumulator{}

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
		if functionCall != nil && !accumulator.Complete {
			accumulator.AddChunk(functionCall)

			// If function call is complete, call the callback
			if accumulator.Complete && onFunctionCall != nil {
				completeFunctionCall := &openai.FunctionCall{
					Name:      accumulator.Name,
					Arguments: accumulator.Arguments,
				}
				onFunctionCall(completeFunctionCall)
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
func sendSSEEvent(reqCtx *gin.Context, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(reqCtx.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
	reqCtx.Writer.Flush()
}

// sendSSEData sends data without an event type
func sendSSEData(reqCtx *gin.Context, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(reqCtx.Writer, "data: %s\n\n", string(jsonData))
	reqCtx.Writer.Flush()
}
