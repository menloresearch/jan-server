package routes

import (
	"github.com/google/wire"
	jan "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan"
	janV1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1"
	janV1Apikeys "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/apikeys"
	janV1Auth "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/auth"
	janV1AuthGoogle "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/auth/google"
	janV1Chat "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/chat"
	janV1Conversations "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/conversations"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/conversations"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp"
	mcp_impl "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp/mcp_impl"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization"
)

var RouteProvider = wire.NewSet(
	janV1Chat.NewCompletionAPI,
	janV1Chat.NewChatRoute,
	janV1Apikeys.NewApiKeyAPI,
	janV1AuthGoogle.NewGoogleAuthAPI,
	janV1Auth.NewAuthRoute,
	janV1Conversations.NewConversationAPI,
	janV1.NewV1Route,
	jan.NewJanRoute,
	organization.NewAdminApiKeyAPI,
	organization.NewOrganizationRoute,
	conversations.NewConversationAPI,
	mcp_impl.NewSerperMCP,
	chat.NewCompletionAPI,
	chat.NewChatRoute,
	mcp.NewMCPAPI,
	v1.NewModelAPI,
	v1.NewV1Route,
)
