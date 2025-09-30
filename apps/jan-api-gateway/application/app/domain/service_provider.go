package domain

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/cron"
	"menlo.ai/jan-api-gateway/app/domain/invite"
	"menlo.ai/jan-api-gateway/app/domain/mcp/serpermcp"
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/response"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

var ServiceProvider = wire.NewSet(
	auth.NewAuthService,
	invite.NewInviteService,
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
	cron.NewService,
	modelprovider.NewService,
	ProvideModelProviderSecret,
)

func ProvideModelProviderSecret() string {
	secret := environment_variables.EnvironmentVariables.MODEL_PROVIDER_SECRET
	if secret == "" {
		secret = environment_variables.EnvironmentVariables.APIKEY_SECRET
	}
	return secret
}
