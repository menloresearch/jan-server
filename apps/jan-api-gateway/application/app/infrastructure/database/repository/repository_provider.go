package repository

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/apikeyrepo"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/conversationrepo"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/itemrepo"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/organizationrepo"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/projectrepo"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/responserepo"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/userrepo"
)

var RepositoryProvider = wire.NewSet(
	organizationrepo.NewOrganizationGormRepository,
	projectrepo.NewProjectGormRepository,
	apikeyrepo.NewApiKeyGormRepository,
	userrepo.NewUserGormRepository,
	conversationrepo.NewConversationGormRepository,
	itemrepo.NewItemGormRepository,
	responserepo.NewResponseGormRepository,
	transaction.NewDatabase,
)
