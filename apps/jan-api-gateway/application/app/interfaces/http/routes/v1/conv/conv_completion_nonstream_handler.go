package conv

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	infrainference "menlo.ai/jan-api-gateway/app/infrastructure/inference"
)

// CompletionNonStreamHandler handles non-streaming completion business logic
type CompletionNonStreamHandler struct {
	multiProvider       *infrainference.MultiProviderInference
	conversationService *conversation.ConversationService
}

// NewCompletionNonStreamHandler creates a new CompletionNonStreamHandler instance
func NewCompletionNonStreamHandler(multiProvider *infrainference.MultiProviderInference, conversationService *conversation.ConversationService) *CompletionNonStreamHandler {
	return &CompletionNonStreamHandler{
		multiProvider:       multiProvider,
		conversationService: conversationService,
	}
}

// CallCompletionAndGetRestResponse calls the inference model and returns a non-streaming REST response
func (uc *CompletionNonStreamHandler) CallCompletionAndGetRestResponse(ctx context.Context, selection infrainference.ProviderSelection, request openai.ChatCompletionRequest) (*ExtendedCompletionResponse, *common.Error) {

	// Call inference provider
	response, err := uc.multiProvider.CreateCompletion(ctx, selection, request)
	if err != nil {
		return nil, common.NewError(err, "c7d8e9f0-g1h2-3456-cdef-789012345678")
	}

	// Convert response
	return uc.ConvertResponse(response), nil
}

// ConvertResponse converts OpenAI response to our extended response
func (uc *CompletionNonStreamHandler) ConvertResponse(response *openai.ChatCompletionResponse) *ExtendedCompletionResponse {
	return &ExtendedCompletionResponse{
		ChatCompletionResponse: *response,
	}
}

// ModifyCompletionResponse modifies the completion response to include item ID and metadata
func (uc *CompletionNonStreamHandler) ModifyCompletionResponse(response *ExtendedCompletionResponse, conv *conversation.Conversation, conversationCreated bool, assistantItem *conversation.Item, askItemID string, completionItemID string, store bool, storeReasoning bool) *ExtendedCompletionResponse {
	// Replace ID with item ID if assistant item exists
	if assistantItem != nil {
		response.ID = assistantItem.PublicID
	}

	// Add metadata if conversation exists
	if conv != nil {
		title := ""
		if conv.Title != nil {
			title = *conv.Title
		}
		response.Metadata = &ResponseMetadata{
			ConversationID:      conv.PublicID,
			ConversationCreated: conversationCreated,
			ConversationTitle:   title,
			AskItemId:           askItemID,
			CompletionItemId:    completionItemID,
			Store:               store,
			StoreReasoning:      storeReasoning,
		}
	}

	return response
}
