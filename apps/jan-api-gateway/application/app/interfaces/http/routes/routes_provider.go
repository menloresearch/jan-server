package routes

import (
	"github.com/google/wire"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
)

var RouteProvider = wire.NewSet(
	chat.NewCompletionAPI,
	chat.NewChatRoute,
	v1.NewModelAPI,
	v1.NewV1Route,
)
