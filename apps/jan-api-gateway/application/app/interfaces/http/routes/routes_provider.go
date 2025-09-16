package routes

import (
	"github.com/google/wire"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/auth/google"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/conversations"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp"
	mcp_impl "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp/mcp_impl"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/invites"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects"
	projectApikeyRoute "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects/api_keys"
)

var RouteProvider = wire.NewSet(
	google.NewGoogleAuthAPI,
	auth.NewAuthRoute,
	projects.NewProjectsRoute,
	organization.NewAdminApiKeyAPI,
	organization.NewOrganizationRoute,
	mcp_impl.NewSerperMCP,
	chat.NewCompletionAPI,
	chat.NewChatRoute,
	mcp.NewMCPAPI,
	v1.NewModelAPI,
	v1.NewV1Route,
	conversations.NewConversationAPI,
	projectApikeyRoute.NewProjectApiKeyRoute,
	invites.NewInvitesRoute,
)
