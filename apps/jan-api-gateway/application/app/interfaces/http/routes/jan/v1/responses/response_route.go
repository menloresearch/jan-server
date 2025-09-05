package responses

import (
	"github.com/gin-gonic/gin"
	handlerresponses "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
)

// ResponseRoute represents the response API routes
type ResponseRoute struct {
	handler *handlerresponses.ResponseHandler
}

// NewResponseRoute creates a new ResponseRoute instance
func NewResponseRoute(handler *handlerresponses.ResponseHandler) *ResponseRoute {
	return &ResponseRoute{
		handler: handler,
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
	responseGroup := router.Group("", middleware.AuthMiddleware(), responseRoute.handler.UserService.RegisteredUserMiddleware())

	responseGroup.POST("", responseRoute.CreateResponse)
	responseGroup.GET("/:response_id", responseRoute.GetResponse)
	responseGroup.DELETE("/:response_id", responseRoute.DeleteResponse)
	responseGroup.POST("/:response_id/cancel", responseRoute.CancelResponse)
	responseGroup.GET("/:response_id/input_items", responseRoute.ListInputItems)
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
// @Description **Response Status:**
// @Description - `completed`: Response generation finished successfully
// @Description - `processing`: Response is being generated
// @Description - `failed`: Response generation failed
// @Description - `cancelled`: Response was cancelled
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body handlerresponses.CreateResponseRequest true "Request payload containing model, input, and generation parameters"
// @Success 200 {object} handlerresponses.Response "Successful response"
// @Success 202 {object} handlerresponses.Response "Response accepted for background processing"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request payload"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 422 {object} handlerresponses.ErrorResponse "Validation error"
// @Failure 429 {object} handlerresponses.ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /jan/v1/responses [post]
func (responseRoute *ResponseRoute) CreateResponse(reqCtx *gin.Context) {
	responseRoute.handler.CreateResponse(reqCtx)
}

// GetResponse retrieves a response by ID
// @Summary Get a response
// @Description Retrieves an LLM response by its ID. Returns the complete response object including input, output, status, and metadata.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Success 200 {object} handlerresponses.Response "Successful response"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 404 {object} handlerresponses.ErrorResponse "Response not found"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /jan/v1/responses/{response_id} [get]
func (responseRoute *ResponseRoute) GetResponse(reqCtx *gin.Context) {
	responseRoute.handler.GetResponse(reqCtx)
}

// DeleteResponse deletes a response by ID
// @Summary Delete a response
// @Description Deletes an LLM response by its ID.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Success 200 {object} handlerresponses.Response "Successful response"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 404 {object} handlerresponses.ErrorResponse "Response not found"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /jan/v1/responses/{response_id} [delete]
func (responseRoute *ResponseRoute) DeleteResponse(reqCtx *gin.Context) {
	responseRoute.handler.DeleteResponse(reqCtx)
}

// CancelResponse cancels a running response
// @Summary Cancel a response
// @Description Cancels a running LLM response that was created with background=true. Only responses that are currently processing can be cancelled.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response to cancel"
// @Success 200 {object} handlerresponses.Response "Response cancelled successfully"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request or response cannot be cancelled"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 404 {object} handlerresponses.ErrorResponse "Response not found"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /jan/v1/responses/{response_id}/cancel [post]
func (responseRoute *ResponseRoute) CancelResponse(reqCtx *gin.Context) {
	responseRoute.handler.CancelResponse(reqCtx)
}

// ListInputItems lists input items for a response
// @Summary List input items
// @Description Retrieves a paginated list of input items for a response. Supports cursor-based pagination for efficient retrieval of large datasets.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Param limit query int false "Maximum number of items to return (default: 20, max: 100)"
// @Param after query string false "Cursor for pagination - return items after this ID"
// @Param before query string false "Cursor for pagination - return items before this ID"
// @Success 200 {object} handlerresponses.ListInputItemsResponse "Successful response with paginated input items"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request or pagination parameters"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 404 {object} handlerresponses.ErrorResponse "Response not found"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /jan/v1/responses/{response_id}/input_items [get]
func (responseRoute *ResponseRoute) ListInputItems(reqCtx *gin.Context) {
	responseRoute.handler.ListInputItems(reqCtx)
}
