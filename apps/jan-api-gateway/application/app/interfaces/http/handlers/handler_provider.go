package handlers

import (
	"github.com/google/wire"
	responseHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/responses"
)

// HandlerProvider provides all HTTP handlers
var HandlerProvider = wire.NewSet(

	responseHandler.NewResponseHandler,
)
