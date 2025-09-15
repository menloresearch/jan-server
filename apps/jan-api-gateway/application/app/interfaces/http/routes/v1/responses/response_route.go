package responses

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/response"
	handlerresponses "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/responses"
)

// CreateResponseRequest represents the request payload for creating a response
type CreateResponseRequest struct {
	Model       string                 `json:"model" binding:"required" example:"gpt-4"`
	Input       map[string]interface{} `json:"input" binding:"required"`
	Generation  map[string]interface{} `json:"generation,omitempty"`
	Stream      bool                   `json:"stream,omitempty" example:"false"`
	Temperature float64                `json:"temperature,omitempty" example:"0.7"`
	MaxTokens   int                    `json:"max_tokens,omitempty" example:"1000"`
}

// ResponseRoute represents the response API routes
type ResponseRoute struct {
	handler         *handlerresponses.ResponseHandler
	authService     *auth.AuthService
	responseService *response.ResponseService
}

// NewResponseRoute creates a new ResponseRoute instance
func NewResponseRoute(handler *handlerresponses.ResponseHandler, authService *auth.AuthService, responseService *response.ResponseService) *ResponseRoute {
	return &ResponseRoute{
		handler:         handler,
		authService:     authService,
		responseService: responseService,
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
	responseGroup.GET(fmt.Sprintf("/:%s", string(response.ResponseContextKeyPublicID)), responseMiddleWare, responseRoute.GetResponse)
	responseGroup.DELETE(fmt.Sprintf("/:%s", string(response.ResponseContextKeyPublicID)), responseMiddleWare, responseRoute.DeleteResponse)
	responseGroup.POST(fmt.Sprintf("/:%s/cancel", string(response.ResponseContextKeyPublicID)), responseMiddleWare, responseRoute.CancelResponse)
	responseGroup.GET(fmt.Sprintf("/:%s/input_items", string(response.ResponseContextKeyPublicID)), responseMiddleWare, responseRoute.ListInputItems)
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
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateResponseRequest true "Request payload containing model, input, and generation parameters"
// @Success 200 "Successful response with embedded fields"
// @Success 202 "Response accepted for background processing with embedded fields"
// @ExampleResponse 200 "Example successful response"
// @Failure 400 "Invalid request payload"
// @Failure 401 "Unauthorized"
// @Failure 422 "Validation error"
// @Failure 429 "Rate limit exceeded"
// @Failure 500 "Internal server error"
// @Router /v1/responses [post]
func (responseRoute *ResponseRoute) CreateResponse(reqCtx *gin.Context) {
	responseRoute.handler.CreateResponse(reqCtx)
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
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Success 200  "Successful response with embedded fields"
// @Failure 400  "Invalid request"
// @Failure 401  "Unauthorized"
// @Failure 404  "Response not found"
// @Failure 500  "Internal server error"
// @Router /v1/responses/{response_id} [get]
func (responseRoute *ResponseRoute) GetResponse(reqCtx *gin.Context) {
	responseRoute.handler.GetResponse(reqCtx)
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
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Success 200  "Successful response with embedded fields"
// @Failure 400  "Invalid request"
// @Failure 401  "Unauthorized"
// @Failure 404  "Response not found"
// @Failure 500  "Internal server error"
// @Router /v1/responses/{response_id} [delete]
func (responseRoute *ResponseRoute) DeleteResponse(reqCtx *gin.Context) {
	responseRoute.handler.DeleteResponse(reqCtx)
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
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response to cancel"
// @Success 200  "Response cancelled successfully with embedded fields"
// @Failure 400  "Invalid request or response cannot be cancelled"
// @Failure 401  "Unauthorized"
// @Failure 404  "Response not found"
// @Failure 500  "Internal server error"
// @Router /v1/responses/{response_id}/cancel [post]
func (responseRoute *ResponseRoute) CancelResponse(reqCtx *gin.Context) {
	responseRoute.handler.CancelResponse(reqCtx)
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
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Unique identifier of the response"
// @Param limit query int false "Maximum number of items to return (default: 20, max: 100)"
// @Param after query string false "Cursor for pagination - return items after this ID"
// @Param before query string false "Cursor for pagination - return items before this ID"
// @Success 200  "Successful response with paginated input items and embedded fields"
// @Failure 400  "Invalid request or pagination parameters"
// @Failure 401  "Unauthorized"
// @Failure 404  "Response not found"
// @Failure 500  "Internal server error"
// @Router /v1/responses/{response_id}/input_items [get]
func (responseRoute *ResponseRoute) ListInputItems(reqCtx *gin.Context) {
	responseRoute.handler.ListInputItems(reqCtx)
}
