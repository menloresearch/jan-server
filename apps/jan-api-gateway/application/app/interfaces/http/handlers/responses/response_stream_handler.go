package responses

import (
	"net/http"

	"github.com/gin-gonic/gin"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

// StreamHandler handles streaming response requests
type StreamHandler struct {
	*ResponseHandler
}

// NewStreamHandler creates a new StreamHandler instance
func NewStreamHandler(responseHandler *ResponseHandler) *StreamHandler {
	return &StreamHandler{
		ResponseHandler: responseHandler,
	}
}

// CreateStreamResponse handles the business logic for creating a streaming response
func (h *StreamHandler) CreateStreamResponse(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest, key string) {
	// Convert response request to chat completion request
	chatCompletionRequest := h.convertToChatCompletionRequest(request)
	if chatCompletionRequest == nil {
		reqCtx.JSON(http.StatusBadRequest, responsetypes.ErrorResponse{
			Code:  "invalid-input-type",
			Error: "unsupported input type for chat completion",
		})
		return
	}

	// Process with Jan inference client for streaming
	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	err := janInferenceClient.CreateChatCompletionStream(reqCtx, key, *chatCompletionRequest)
	if err != nil {
		reqCtx.AbortWithStatusJSON(
			http.StatusBadRequest,
			responsetypes.ErrorResponse{
				Code:  "c3af973c-eada-4e8b-96d9-e92546588cd3",
				Error: err.Error(),
			})
		return
	}
}
