package domain

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/mcp/serpermcp"
	"menlo.ai/jan-api-gateway/app/domain/user"
)

var ServiceProvider = wire.NewSet(
	apikey.NewService,
	user.NewService,
	serpermcp.NewSerperService,
)
