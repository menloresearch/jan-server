package handlers

import (
	"github.com/google/wire"
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
	responseHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/responses"
)

// HandlerProvider provides all HTTP handlers
var HandlerProvider = wire.NewSet(

	responseHandler.NewResponseHandler,
	conversationHandler.NewConversationHandler,
)
