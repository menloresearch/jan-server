package routes

import (
	"github.com/google/wire"
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
	jan "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan"
	janPlatformV1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan-platform/v1"
	janPlatformV1Organization "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan-platform/v1/organization"
	janPlatformV1OrganizationProject "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan-platform/v1/organization/projects"
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
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects"
)

var RouteProvider = wire.NewSet(
	janPlatformV1.NewV1Route,
	janPlatformV1OrganizationProject.NewProjectsRoute,
	janPlatformV1Organization.NewOrganizationRoute,
	janV1Chat.NewCompletionAPI,
	janV1Chat.NewChatRoute,
	janV1Apikeys.NewApiKeyAPI,
	janV1AuthGoogle.NewGoogleAuthAPI,
	janV1Auth.NewAuthRoute,
	janV1.NewV1Route,
	jan.NewJanRoute,
	projects.NewProjectsRoute,
	organization.NewAdminApiKeyAPI,
	organization.NewOrganizationRoute,
	mcp_impl.NewSerperMCP,
	chat.NewCompletionAPI,
	chat.NewChatRoute,
	mcp.NewMCPAPI,
	v1.NewModelAPI,
	v1.NewV1Route,

	// Conversation-related dependencies
	conversationHandler.NewConversationHandler,
	conversations.NewConversationAPI,
	janV1Conversations.NewConversationAPI,
)
