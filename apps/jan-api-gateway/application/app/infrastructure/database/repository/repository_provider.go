package repository

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/userrepo"
)

var RepositoryProvider = wire.NewSet(
	userrepo.NewUserGormRepository,
)
