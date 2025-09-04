package responses

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/user"
	handlerresponses "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
)

// ResponseRoute represents the response API routes
type ResponseRoute struct {
	handler *handlerresponses.ResponseHandler
}

// NewResponseRoute creates a new ResponseRoute instance
func NewResponseRoute(
	userService *user.UserService,
	apikeyService *apikey.ApiKeyService,
) *ResponseRoute {
	return &ResponseRoute{
		handler: handlerresponses.NewResponseHandler(userService, apikeyService),
	}
}

// RegisterRouter registers the response routes
func (responseRoute *ResponseRoute) RegisterRouter(router gin.IRouter) {
	responseRouter := router.Group("/responses")
	responseRoute.registerRoutes(responseRouter)
}

// registerRoutes registers all response routes
func (responseRoute *ResponseRoute) registerRoutes(router *gin.RouterGroup) {
	// Create Response route: POST https://api.openai.com/v1/responses
	router.POST("", middleware.OptionalAuthMiddleware(), responseRoute.handler.UserService.RegisteredUserMiddleware(), responseRoute.CreateResponse)

	// Get Response route: GET https://api.openai.com/v1/responses/{response_id}
	router.GET("/:response_id", middleware.OptionalAuthMiddleware(), responseRoute.handler.UserService.RegisteredUserMiddleware(), responseRoute.GetResponse)

	// Delete Response route: DELETE https://api.openai.com/v1/responses/{response_id}
	router.DELETE("/:response_id", middleware.OptionalAuthMiddleware(), responseRoute.handler.UserService.RegisteredUserMiddleware(), responseRoute.DeleteResponse)

	// Cancel Response route: POST https://api.openai.com/v1/responses/{response_id}/cancel
	router.POST("/:response_id/cancel", middleware.OptionalAuthMiddleware(), responseRoute.handler.UserService.RegisteredUserMiddleware(), responseRoute.CancelResponse)

	// List Input Items route: GET https://api.openai.com/v1/responses/{response_id}/input_items
	router.GET("/:response_id/input_items", middleware.OptionalAuthMiddleware(), responseRoute.handler.UserService.RegisteredUserMiddleware(), responseRoute.ListInputItems)
}

// CreateResponse handles the POST /v1/responses endpoint
// @Summary Create a response
// @Description Creates a model response for the given input.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body handlerresponses.CreateResponseRequest true "Create response request payload"
// @Success 200 {object} handlerresponses.Response "Successful response"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request payload"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /v1/responses [post]
func (responseRoute *ResponseRoute) CreateResponse(reqCtx *gin.Context) {
	responseRoute.handler.CreateResponse(reqCtx)
}

// GetResponse handles the GET /v1/responses/{response_id} endpoint
// @Summary Get a response
// @Description Retrieves a model response by its ID.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} handlerresponses.Response "Successful response"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 404 {object} handlerresponses.ErrorResponse "Response not found"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /v1/responses/{response_id} [get]
func (responseRoute *ResponseRoute) GetResponse(reqCtx *gin.Context) {
	responseRoute.handler.GetResponse(reqCtx)
}

// DeleteResponse handles the DELETE /v1/responses/{response_id} endpoint
// @Summary Delete a response
// @Description Deletes a model response by its ID.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} handlerresponses.Response "Successful response"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 404 {object} handlerresponses.ErrorResponse "Response not found"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /v1/responses/{response_id} [delete]
func (responseRoute *ResponseRoute) DeleteResponse(reqCtx *gin.Context) {
	responseRoute.handler.DeleteResponse(reqCtx)
}

// CancelResponse handles the POST /v1/responses/{response_id}/cancel endpoint
// @Summary Cancel a response
// @Description Cancels a response created with the background parameter set to true.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} handlerresponses.Response "Successful response"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 404 {object} handlerresponses.ErrorResponse "Response not found"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /v1/responses/{response_id}/cancel [post]
func (responseRoute *ResponseRoute) CancelResponse(reqCtx *gin.Context) {
	responseRoute.handler.CancelResponse(reqCtx)
}

// ListInputItems handles the GET /v1/responses/{response_id}/input_items endpoint
// @Summary List input items
// @Description Retrieves a list of input items for a given response ID.
// @Tags Jan, Jan-Responses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param response_id path string true "Response ID"
// @Param limit query int false "Number of items to return"
// @Param after query string false "Cursor for pagination"
// @Param before query string false "Cursor for pagination"
// @Success 200 {object} handlerresponses.ListInputItemsResponse "Successful response"
// @Failure 400 {object} handlerresponses.ErrorResponse "Invalid request"
// @Failure 401 {object} handlerresponses.ErrorResponse "Unauthorized"
// @Failure 404 {object} handlerresponses.ErrorResponse "Response not found"
// @Failure 500 {object} handlerresponses.ErrorResponse "Internal server error"
// @Router /v1/responses/{response_id}/input_items [get]
func (responseRoute *ResponseRoute) ListInputItems(reqCtx *gin.Context) {
	responseRoute.handler.ListInputItems(reqCtx)
}
