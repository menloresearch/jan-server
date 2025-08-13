package domain

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/domain/mcp/serpermcp"
)

var ServiceProvider = wire.NewSet(
	serpermcp.NewSerperService,
)
