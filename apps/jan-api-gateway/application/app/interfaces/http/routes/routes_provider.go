package routes

import (
	"github.com/google/wire"
	admin "menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin"
	adminV1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin/v1"
	adminV1Apikeys "menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin/v1/apikeys"
	adminV1Auth "menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin/v1/auth"
	adminV1AuthGoogle "menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin/v1/auth/google"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp"
	mcp_impl "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp/mcp_impl"
)

var RouteProvider = wire.NewSet(
	adminV1Apikeys.NewApiKeyAPI,
	adminV1AuthGoogle.NewGoogleAuthAPI,
	adminV1Auth.NewAuthRoute,
	adminV1.NewV1Route,
	admin.NewAdminRoute,
	mcp_impl.NewSerperMCP,
	chat.NewCompletionAPI,
	chat.NewChatRoute,
	mcp.NewMCPAPI,
	v1.NewModelAPI,
	v1.NewV1Route,
)
