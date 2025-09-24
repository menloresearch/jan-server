package routes

import (
	"github.com/google/wire"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/auth/google"
	chat "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
	conv_chat "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/conv"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/conversations"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp"
	mcp_impl "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp/mcp_impl"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/invites"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects"
	api_keys "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects/api_keys"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/responses"
)

var RouteProvider = wire.NewSet(
	google.NewGoogleAuthAPI,
	auth.NewAuthRoute,
	projects.NewProjectsRoute,
	organization.NewAdminApiKeyAPI,
	organization.NewOrganizationRoute,
	mcp_impl.NewSerperMCP,
	chat.NewChatRoute,
	chat.NewCompletionAPI,
	conv_chat.NewConvChatRoute,
	conv_chat.NewConvCompletionAPI,
	conv_chat.NewConvMCPAPI,
	conv_chat.NewCompletionNonStreamHandler,
	conv_chat.NewCompletionStreamHandler,
	mcp.NewMCPAPI,
	v1.NewModelAPI,
	responses.NewResponseRoute,
	v1.NewV1Route,
	conversations.NewConversationAPI,
	invites.NewInvitesRoute,
	api_keys.NewProjectApiKeyRoute,
)
