package responses

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/response"

	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

// Use types from the response packages instead of defining internal types

// ResponseRoute represents the response API routes
type ResponseRoute struct {
	responseModelService  *response.ResponseModelService
	authService           *auth.AuthService
	responseService       *response.ResponseService
	streamModelService    *response.StreamModelService
	nonStreamModelService *response.NonStreamModelService
}

// NewResponseRoute creates a new ResponseRoute instance
func NewResponseRoute(responseModelService *response.ResponseModelService, authService *auth.AuthService, responseService *response.ResponseService, streamHandler *response.StreamModelService, nonStreamHandler *response.NonStreamModelService) *ResponseRoute {
	return &ResponseRoute{
		responseModelService:  responseModelService,
		authService:           authService,
		responseService:       responseService,
		streamModelService:    streamHandler,
		nonStreamModelService: nonStreamHandler,
	}
}

// RegisterRouter registers the response routes
func (responseRoute *ResponseRoute) RegisterRouter(router gin.IRouter) {
	responseRouter := router.Group("/responses")
	responseRoute.registerRoutes(responseRouter)
}

// registerRoutes registers all response routes
func (responseRoute *ResponseRoute) registerRoutes(router *gin.RouterGroup) {
	// Apply middleware to the entire group
	responseGroup := router.Group("",
		responseRoute.authService.AppUserAuthMiddleware(),
		responseRoute.authService.RegisteredUserMiddleware(),
	)

	responseGroup.POST("", responseRoute.CreateResponse)

	// Apply response middleware for routes that need response context
	responseMiddleWare := responseRoute.responseService.GetResponseMiddleWare()
	responseGroup.GET(fmt.Sprintf("/:%s", string(response.ResponseContextKeyPublicID)), responseMiddleWare, responseRoute.responseModelService.GetResponseHandler)
	responseGroup.DELETE(fmt.Sprintf("/:%s", string(response.ResponseContextKeyPublicID)), responseMiddleWare, responseRoute.responseModelService.DeleteResponseHandler)
	responseGroup.POST(fmt.Sprintf("/:%s/cancel", string(response.ResponseContextKeyPublicID)), responseMiddleWare, responseRoute.responseModelService.CancelResponseHandler)
	responseGroup.GET(fmt.Sprintf("/:%s/input_items", string(response.ResponseContextKeyPublicID)), responseMiddleWare, responseRoute.responseModelService.ListInputItemsHandler)
}

// CreateResponse creates a new response from LLM
// @Summary Create a response
// @Description Creates a new LLM response for the given input. Supports multiple input types including text, images, files, web search, and more.
// @Description
// @Description **Supported Input Types:**
// @Description - `text`: Plain text input
// @Description - `image`: Image input (URL or base64)
// @Description - `file`: File input by file ID
// @Description - `web_search`: Web search input
// @Description - `file_search`: File search input
// @Description - `streaming`: Streaming input
// @Description - `function_calls`: Function calls input
// @Description - `reasoning`: Reasoning input
// @Description
// @Description **Example Request:**
// @Description ```json
// @Description {
// @Description   "model": "gpt-4",
// @Description   "input": {
// @Description     "type": "text",
// @Description     "text": "Hello, how are you?"
// @Description   },
// @Description   "max_tokens": 100,
// @Description   "temperature": 0.7,
// @Description   "stream": false,
// @Description   "background": false
// @Description }
// @Description ```
// @Description
// @Description **Response Format:**
// @Description The response uses embedded structure where all fields are at the top level:
// @Description - `jan_status`: Jan API status code (optional)
// @Description - `id`: Response identifier
// @Description - `object`: Object type ("response")
// @Description - `created`: Unix timestamp
// @Description - `model`: Model used
// @Description - `status`: Response status
// @Description - `input`: Input data
// @Description - `output`: Generated output
// @Description
// @Description **Example Response:**
// @Description ```json
// @Description {
// @Description   "jan_status": "000000",
// @Description   "id": "resp_1234567890",
// @Description   "object": "response",
// @Description   "created": 1234567890,
// @Description   "model": "gpt-4",
// @Description   "status": "completed",
// @Description   "input": {
// @Description     "type": "text",
// @Description     "text": "Hello, how are you?"
// @Description   },
// @Description   "output": {
// @Description     "type": "text",
// @Description     "text": {
// @Description       "value": "I'm doing well, thank you!"
// @Description     }
// @Description   }
// @Description }
// @Description ```
// @Description
// @Description **Response Status:**
// @Description - `completed`: Response generation finished successfully
// @Description - `processing`: Response is being generated
// @Description - `failed`: Response generation failed
// @Description - `cancelled`: Response was cancelled
// @Tags Responses API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body requesttypes.CreateResponseRequest true "Request payload containing model, input, and generation parameters"
// @Success 200 {object} responses.Response "Created response"
// @Success 202 {object} responses.Response "Response accepted for background processing"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 422 {object} responses.ErrorResponse "Validation error"
// @Failure 429 {object} responses.ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/responses [post]
func (responseRoute *ResponseRoute) CreateResponse(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	user, _ := auth.GetUserFromContext(reqCtx)
	userID := user.ID

	var request requesttypes.CreateResponseRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "g7h8i9j0-k1l2-3456-ghij-789012345678",
		})
		return
	}

	// Validate request parameters
	if request.Model == "" {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "h8i9j0k1-l2m3-4567-hijk-890123456789",
		})
		return
	}

	if request.Input == nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "i9j0k1l2-m3n4-5678-ijkl-901234567890",
		})
		return
	}

	// Convert to domain request type
	domainRequest := &requesttypes.CreateResponseRequest{
		Model:              request.Model,
		Input:              request.Input,
		Stream:             request.Stream,
		Temperature:        request.Temperature,
		MaxTokens:          request.MaxTokens,
		PreviousResponseID: request.PreviousResponseID,
		SystemPrompt:       request.SystemPrompt,
		TopP:               request.TopP,
		TopK:               request.TopK,
		RepetitionPenalty:  request.RepetitionPenalty,
		Seed:               request.Seed,
		Stop:               request.Stop,
		PresencePenalty:    request.PresencePenalty,
		FrequencyPenalty:   request.FrequencyPenalty,
		LogitBias:          request.LogitBias,
		ResponseFormat:     request.ResponseFormat,
		Tools:              request.Tools,
		ToolChoice:         request.ToolChoice,
		Metadata:           request.Metadata,
		Background:         request.Background,
		Timeout:            request.Timeout,
		User:               request.User,
		Conversation:       request.Conversation,
		Store:              request.Store,
	}

	// Call domain service (pure business logic)
	result, err := responseRoute.responseModelService.CreateResponse(ctx, userID, domainRequest)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}

	// Handle HTTP/SSE concerns directly
	responseRoute.handleResponseCreation(reqCtx, result, domainRequest)
}

// handleResponseCreation handles both streaming and non-streaming response creation
func (responseRoute *ResponseRoute) handleResponseCreation(reqCtx *gin.Context, result *response.ResponseCreationResult, request *requesttypes.CreateResponseRequest) {
	// Set up streaming headers if needed
	if result.IsStreaming {
		reqCtx.Header("Content-Type", "text/event-stream")
		reqCtx.Header("Cache-Control", "no-cache")
		reqCtx.Header("Connection", "keep-alive")
		reqCtx.Header("Access-Control-Allow-Origin", "*")
		reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")
	}

	// Delegate to appropriate handler based on streaming preference
	if result.IsStreaming {
		responseRoute.streamModelService.CreateStreamResponse(reqCtx, request, result.APIKey, result.Conversation, result.Response, result.ChatCompletionRequest)
	} else {
		responseRoute.nonStreamModelService.CreateNonStreamResponseHandler(reqCtx, request, result.APIKey, result.Conversation, result.Response, result.ChatCompletionRequest)
	}
}

// GetResponse retrieves a response by ID
// @Summary Get a response
// @Description Retrieves an LLM response by its ID. Returns the complete response object with embedded structure where all fields are at the top level.
// @Description
// @Description **Response Format:**
// @Description The response uses embedded structure where all fields are at the top level:
// @Description - `jan_status`: Jan API status code (optional)
// @Description - `id`: Response identifier
// @Description - `object`: Object type ("response")
// @Description - `created`: Unix timestamp
// @Description - `model`: Model used
// @Description - `status`: Response status
// @Description - `input`: Input data
// @Description - `output`: Generated output
// @Tags Responses API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Success 200 {object} responses.Response "Response details"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Response not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/responses/{response_id} [get]
func (responseRoute *ResponseRoute) GetResponse(reqCtx *gin.Context) {
	resp, ok := response.GetResponseFromContext(reqCtx)
	if !ok {
		return
	}
	// Convert domain response to API response using the service
	apiResponse := responseRoute.responseService.ConvertDomainResponseToAPIResponse(resp)
	reqCtx.JSON(http.StatusOK, apiResponse)
}

// DeleteResponse deletes a response by ID
// @Summary Delete a response
// @Description Deletes an LLM response by its ID. Returns the deleted response object with embedded structure where all fields are at the top level.
// @Description
// @Description **Response Format:**
// @Description The response uses embedded structure where all fields are at the top level:
// @Description - `jan_status`: Jan API status code (optional)
// @Description - `id`: Response identifier
// @Description - `object`: Object type ("response")
// @Description - `created`: Unix timestamp
// @Description - `model`: Model used
// @Description - `status`: Response status (will be "cancelled")
// @Description - `input`: Input data
// @Description - `cancelled_at`: Cancellation timestamp
// @Tags Responses API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Success 200 {object} responses.Response "Deleted response"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Response not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/responses/{response_id} [delete]
func (responseRoute *ResponseRoute) DeleteResponse(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	resp, ok := response.GetResponseFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "k1l2m3n4-o5p6-7890-klmn-123456789012",
		})
		return
	}

	success, err := responseRoute.responseService.DeleteResponse(ctx, resp.ID)
	if !success {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}
	// Convert domain response to API response using the service
	apiResponse := responseRoute.responseService.ConvertDomainResponseToAPIResponse(resp)
	reqCtx.JSON(http.StatusOK, apiResponse)
}

// CancelResponse cancels a running response
// @Summary Cancel a response
// @Description Cancels a running LLM response that was created with background=true. Only responses that are currently processing can be cancelled.
// @Description
// @Description **Response Format:**
// @Description The response uses embedded structure where all fields are at the top level:
// @Description - `jan_status`: Jan API status code (optional)
// @Description - `id`: Response identifier
// @Description - `object`: Object type ("response")
// @Description - `created`: Unix timestamp
// @Description - `model`: Model used
// @Description - `status`: Response status (will be "cancelled")
// @Description - `input`: Input data
// @Description - `cancelled_at`: Cancellation timestamp
// @Tags Responses API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response to cancel"
// @Success 200 {object} responses.Response "Response cancelled successfully"
// @Failure 400 {object} responses.ErrorResponse "Invalid request or response cannot be cancelled"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Response not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/responses/{response_id}/cancel [post]
func (responseRoute *ResponseRoute) CancelResponse(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	resp, ok := response.GetResponseFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "m3n4o5p6-q7r8-9012-mnop-345678901234",
		})
		return
	}

	// TODO
	// Cancel the stream if it is streaming in go routine and update response status in go routine
	success, err := responseRoute.responseService.UpdateResponseStatus(ctx, resp.ID, response.ResponseStatusCancelled)
	if !success {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}

	// Reload the response to get updated status
	updatedResp, err := responseRoute.responseService.GetResponseByPublicID(ctx, resp.PublicID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}
	// Convert domain response to API response using the service
	apiResponse := responseRoute.responseService.ConvertDomainResponseToAPIResponse(updatedResp)
	reqCtx.JSON(http.StatusOK, apiResponse)
}

// ListInputItems lists input items for a response
// @Summary List input items
// @Description Retrieves a paginated list of input items for a response. Supports cursor-based pagination for efficient retrieval of large datasets.
// @Description
// @Description **Response Format:**
// @Description The response uses embedded structure where all fields are at the top level:
// @Description - `jan_status`: Jan API status code (optional)
// @Description - `first_id`: First item ID for pagination (optional)
// @Description - `last_id`: Last item ID for pagination (optional)
// @Description - `has_more`: Whether more items are available (optional)
// @Description - `id`: Input item identifier
// @Description - `object`: Object type ("input_item")
// @Description - `created`: Unix timestamp
// @Description - `type`: Input type
// @Description - `text`: Text content (for text type)
// @Description - `image`: Image content (for image type)
// @Description - `file`: File content (for file type)
// @Description
// @Description **Example Response:**
// @Description ```json
// @Description {
// @Description   "jan_status": "000000",
// @Description   "first_id": "input_123",
// @Description   "last_id": "input_456",
// @Description   "has_more": false,
// @Description   "id": "input_1234567890",
// @Description   "object": "input_item",
// @Description   "created": 1234567890,
// @Description   "type": "text",
// @Description   "text": "Hello, world!"
// @Description }
// @Description ```
// @Tags Responses API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Param limit query int false "Maximum number of items to return (default: 20, max: 100)"
// @Param after query string false "Cursor for pagination - return items after this ID"
// @Param before query string false "Cursor for pagination - return items before this ID"
// @Success 200 {object} responses.ListInputItemsResponse "List of input items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request or pagination parameters"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Response not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/responses/{response_id}/input_items [get]
func (responseRoute *ResponseRoute) ListInputItems(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	resp, ok := response.GetResponseFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "p6q7r8s9-t0u1-2345-pqrs-678901234567",
		})
		return
	}

	// Get items for this response using the response service
	items, err := responseRoute.responseService.GetItemsForResponse(ctx, resp.ID, nil)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}

	var firstId *string
	var lastId *string
	if len(items) > 0 {
		firstId = &items[0].PublicID
		lastId = &items[len(items)-1].PublicID
	}

	// Convert conversation items to input items using the service
	inputItems := make([]responses.InputItem, 0, len(items))
	for _, item := range items {
		inputItem := responseRoute.responseService.ConvertConversationItemToInputItem(item)
		inputItems = append(inputItems, inputItem)
	}

	reqCtx.JSON(http.StatusOK, responses.ListInputItemsResponse{
		Object:  "list",
		Data:    inputItems,
		FirstID: firstId,
		LastID:  lastId,
		HasMore: false, // For now, we'll return all items without pagination
	})
}

// All transformation functions removed - now using service methods
