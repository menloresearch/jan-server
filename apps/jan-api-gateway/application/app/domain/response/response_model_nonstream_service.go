package response

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

const (
	// DefaultTimeout is the default timeout for non-streaming requests
	DefaultTimeout = 120 * time.Second
)

// NonStreamModelService handles non-streaming response requests
type NonStreamModelService struct {
	*ResponseModelService
}

// NewNonStreamModelService creates a new NonStreamModelService instance
func NewNonStreamModelService(responseModelService *ResponseModelService) *NonStreamModelService {
	return &NonStreamModelService{
		ResponseModelService: responseModelService,
	}
}

// CreateNonStreamResponse handles the business logic for creating a non-streaming response
func (h *NonStreamModelService) CreateNonStreamResponseHandler(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest, key string, conv *conversation.Conversation, responseEntity *Response, chatCompletionRequest *openai.ChatCompletionRequest) {

	result, err := h.CreateNonStreamResponse(reqCtx, request, key, conv, responseEntity, chatCompletionRequest)
	if err != nil {
		reqCtx.AbortWithStatusJSON(
			http.StatusBadRequest,
			responsetypes.ErrorResponse{
				Code:  err.GetCode(),
				Error: err.Error(),
			})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doCreateNonStreamResponse performs the business logic for creating a non-streaming response
func (h *NonStreamModelService) CreateNonStreamResponse(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest, key string, conv *conversation.Conversation, responseEntity *Response, chatCompletionRequest *openai.ChatCompletionRequest) (responsetypes.Response, *common.Error) {
	// Process with Jan inference client for non-streaming with timeout
	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), DefaultTimeout)
	defer cancel()
	chatResponse, err := janInferenceClient.CreateChatCompletion(ctx, key, *chatCompletionRequest)
	if err != nil {
		return responsetypes.Response{}, common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
	}

	// Process reasoning content
	var processedResponse *openai.ChatCompletionResponse = chatResponse

	// Append assistant's response to conversation (only if conversation exists)
	if conv != nil && len(processedResponse.Choices) > 0 && processedResponse.Choices[0].Message.Content != "" {
		assistantMessage := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: processedResponse.Choices[0].Message.Content,
		}
		success, err := h.responseService.AppendMessagesToConversation(reqCtx, conv, []openai.ChatCompletionMessage{assistantMessage}, &responseEntity.ID)
		if !success {
			// Log error but don't fail the response
			logger.GetLogger().Errorf("Failed to append assistant response to conversation: %s - %s", err.GetCode(), err.Error())
		}
	}

	// Convert chat completion response to response format
	responseData := h.convertFromChatCompletionResponse(processedResponse, request, conv, responseEntity)

	// Update response with all fields at once (optimized to prevent N+1 queries)
	updates := &ResponseUpdates{
		Status: ptr.ToString(string(ResponseStatusCompleted)),
		Output: responseData.T.Output,
		Usage:  responseData.T.Usage,
	}
	success, updateErr := h.responseService.UpdateResponseFields(reqCtx, responseEntity.ID, updates)
	if !success {
		// Log error but don't fail the request since response is already generated
		logger.GetLogger().Errorf("Failed to update response fields: %s - %s\n", updateErr.GetCode(), updateErr.Error())
	}

	return responseData.T, nil
}

// convertFromChatCompletionResponse converts a ChatCompletionResponse to a Response
func (h *NonStreamModelService) convertFromChatCompletionResponse(chatResp *openai.ChatCompletionResponse, req *requesttypes.CreateResponseRequest, conv *conversation.Conversation, responseEntity *Response) responsetypes.OpenAIGeneralResponse[responsetypes.Response] {

	// Extract the content and reasoning from the first choice
	var outputText string
	var reasoningContent string

	if len(chatResp.Choices) > 0 {
		choice := chatResp.Choices[0]
		outputText = choice.Message.Content

		// Extract reasoning content if present
		if choice.Message.ReasoningContent != "" {
			reasoningContent = choice.Message.ReasoningContent
		}
	}

	// Convert input back to the original format for response
	var responseInput any
	switch v := req.Input.(type) {
	case string:
		responseInput = v
	case []any:
		responseInput = v
	default:
		responseInput = req.Input
	}

	// Create output using proper ResponseOutput structure
	var output []responsetypes.ResponseOutput

	// Add reasoning content if present
	if reasoningContent != "" {
		output = append(output, responsetypes.ResponseOutput{
			Type: responsetypes.OutputTypeReasoning,
			Reasoning: &responsetypes.ReasoningOutput{
				Task:   "reasoning",
				Result: reasoningContent,
				Steps:  []responsetypes.ReasoningStep{},
			},
		})
	}

	// Add text content if present
	if outputText != "" {
		output = append(output, responsetypes.ResponseOutput{
			Type: responsetypes.OutputTypeText,
			Text: &responsetypes.TextOutput{
				Value:       outputText,
				Annotations: []responsetypes.Annotation{},
			},
		})
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
		ID:           responseEntity.PublicID,
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
			Effort: nil,
			Summary: func() *string {
				if reasoningContent != "" {
					return &reasoningContent
				}
				return nil
			}(),
		},
		Store:       true,
		Temperature: req.Temperature,
		Text: &responsetypes.TextFormat{
			Format: &responsetypes.FormatType{
				Type: "text",
			},
		},
		TopP:       req.TopP,
		Truncation: "disabled",
		User:       nil,
		Metadata:   req.Metadata,
	}

	return responsetypes.OpenAIGeneralResponse[responsetypes.Response]{
		T: response,
	}
}
