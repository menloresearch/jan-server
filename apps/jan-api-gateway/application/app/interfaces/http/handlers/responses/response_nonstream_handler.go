package responses

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

const (
	// DefaultTimeout is the default timeout for non-streaming requests
	DefaultTimeout = 60 * time.Second
)

// NonStreamHandler handles non-streaming response requests
type NonStreamHandler struct {
	*ResponseHandler
}

// NewNonStreamHandler creates a new NonStreamHandler instance
func NewNonStreamHandler(responseHandler *ResponseHandler) *NonStreamHandler {
	return &NonStreamHandler{
		ResponseHandler: responseHandler,
	}
}

// CreateNonStreamResponse handles the business logic for creating a non-streaming response
func (h *NonStreamHandler) CreateNonStreamResponse(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest, key string, conv *conversation.Conversation) {
	// Convert response request to chat completion request
	chatCompletionRequest := h.convertToChatCompletionRequest(request)
	if chatCompletionRequest == nil {
		reqCtx.JSON(http.StatusBadRequest, responsetypes.ErrorResponse{
			Code:  "019929ec-6f89-76c5-8ed4-bd0eb1c6c8db",
			Error: "unsupported input type for chat completion",
		})
		return
	}

	// Process with Jan inference client for non-streaming with timeout
	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), DefaultTimeout)
	defer cancel()
	response, err := janInferenceClient.CreateChatCompletion(ctx, key, *chatCompletionRequest)
	if err != nil {
		reqCtx.AbortWithStatusJSON(
			http.StatusBadRequest,
			responsetypes.ErrorResponse{
				Code:  "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4",
				Error: err.Error(),
			})
		return
	}

	// Append assistant's response to conversation
	if len(response.Choices) > 0 && response.Choices[0].Message.Content != "" {
		assistantMessage := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: response.Choices[0].Message.Content,
		}
		if err := h.appendMessagesToConversation(reqCtx, conv, []openai.ChatCompletionMessage{assistantMessage}); err != nil {
			// Log error but don't fail the response
			logger.GetLogger().Errorf("Failed to append assistant response to conversation: %v", err)
		}
	}

	// Convert chat completion response to response format
	responseData := h.convertFromChatCompletionResponse(response, request, conv)
	reqCtx.JSON(http.StatusOK, responseData.T)
}

// convertFromChatCompletionResponse converts a ChatCompletionResponse to a Response
func (h *NonStreamHandler) convertFromChatCompletionResponse(chatResp *openai.ChatCompletionResponse, req *requesttypes.CreateResponseRequest, conv *conversation.Conversation) responsetypes.OpenAIGeneralResponse[responsetypes.Response] {

	// Extract the content from the first choice
	var outputText string
	if len(chatResp.Choices) > 0 {
		outputText = chatResp.Choices[0].Message.Content
	}

	// Convert input back to the original format for response
	var responseInput interface{}
	switch v := req.Input.(type) {
	case string:
		responseInput = v
	case []interface{}:
		responseInput = v
	default:
		responseInput = req.Input
	}

	// Create output using proper ResponseOutput structure
	output := []responsetypes.ResponseOutput{
		{
			Type: responsetypes.OutputTypeText,
			Text: &responsetypes.TextOutput{
				Value:       outputText,
				Annotations: []responsetypes.Annotation{},
			},
		},
	}

	// Create usage information using proper DetailedUsage struct
	usage := &responsetypes.DetailedUsage{
		InputTokens:  chatResp.Usage.PromptTokens,
		OutputTokens: chatResp.Usage.CompletionTokens,
		TotalTokens:  chatResp.Usage.TotalTokens,
		InputTokensDetails: &responsetypes.TokenDetails{
			CachedTokens: 0,
		},
		OutputTokensDetails: &responsetypes.TokenDetails{
			ReasoningTokens: 0,
		},
	}

	// Create conversation info
	var conversationInfo *responsetypes.ConversationInfo
	if conv != nil {
		conversationInfo = &responsetypes.ConversationInfo{
			ID: conv.PublicID,
		}
	}

	response := responsetypes.Response{
		ID:           chatResp.ID,
		Object:       "response",
		Created:      chatResp.Created,
		Model:        chatResp.Model,
		Status:       responsetypes.ResponseStatusCompleted,
		Input:        responseInput,
		Output:       output,
		Usage:        usage,
		Conversation: conversationInfo,
		// Add other OpenAI response fields
		Error:              nil,
		IncompleteDetails:  nil,
		Instructions:       nil,
		MaxOutputTokens:    req.MaxTokens,
		ParallelToolCalls:  false,
		PreviousResponseID: nil,
		Reasoning: &responsetypes.Reasoning{
			Effort:  nil,
			Summary: nil,
		},
		Store:       true,
		Temperature: req.Temperature,
		Text: &responsetypes.TextFormat{
			Format: &responsetypes.FormatType{
				Type: "text",
			},
		},
		ToolChoice: &requesttypes.ToolChoice{
			Type: "auto",
		},
		Tools:      []requesttypes.Tool{},
		TopP:       req.TopP,
		Truncation: "disabled",
		User:       nil,
		Metadata:   req.Metadata,
	}

	return responsetypes.OpenAIGeneralResponse[responsetypes.Response]{
		T: response,
	}
}
