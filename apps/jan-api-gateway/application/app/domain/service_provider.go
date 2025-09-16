package domain

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/mcp/serpermcp"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/response"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
)

var ServiceProvider = wire.NewSet(
	auth.NewAuthService,
	organization.NewService,
	project.NewService,
	apikey.NewService,
	user.NewService,
	conversation.NewService,
	response.NewResponseService,
	response.NewResponseModelService,
	response.NewStreamModelService,
	response.NewNonStreamModelService,
	serpermcp.NewSerperService,
	chat.NewChatUseCase,
	chat.NewStreamingService,
)
