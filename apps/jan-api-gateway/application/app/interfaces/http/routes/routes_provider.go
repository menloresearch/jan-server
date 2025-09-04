package routes

import (
	"github.com/google/wire"
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
	jan "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan"
	janV1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1"
	janV1Auth "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/auth"
	janV1AuthGoogle "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/auth/google"
	janV1Chat "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/chat"
	janV1Conversations "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/conversations"
	janV1Organization "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization"
	janV1OrganizationApikeys "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization/api_keys"
	janV1OrganizationProject "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization/projects"
	janV1OrganizationProjectApikeys "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization/projects/api_keys"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/conversations"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp"
	mcp_impl "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp/mcp_impl"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects"
)

var RouteProvider = wire.NewSet(
	janV1OrganizationApikeys.NewOrganizationApiKeyRouteRoute,
	janV1OrganizationProjectApikeys.NewProjectApiKeyRoute,
	janV1.NewV1Route,
	janV1OrganizationProject.NewProjectsRoute,
	janV1Organization.NewOrganizationRoute,
	janV1Chat.NewCompletionAPI,
	janV1Chat.NewChatRoute,
	janV1AuthGoogle.NewGoogleAuthAPI,
	janV1Auth.NewAuthRoute,
	conversationHandler.NewConversationHandler,
	janV1Conversations.NewConversationAPI,
	janV1.NewV1Route,
	jan.NewJanRoute,
	projects.NewProjectsRoute,
	organization.NewAdminApiKeyAPI,
	organization.NewOrganizationRoute,
	mcp_impl.NewSerperMCP,
	chat.NewCompletionAPI,
	chat.NewChatRoute,
	conversations.NewConversationAPI,
	mcp.NewMCPAPI,
	v1.NewModelAPI,
	v1.NewV1Route,
)
