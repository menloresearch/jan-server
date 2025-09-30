package cron

import (
	"context"

	"github.com/mileusna/crontab"
	infrainference "menlo.ai/jan-api-gateway/app/infrastructure/inference"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type JanModelRefresher interface {
	RefreshModels(ctx context.Context) (*infrainference.ModelsResponse, error)
}

type CronService struct {
	JanProvider JanModelRefresher
}

func NewService(janProvider JanModelRefresher) *CronService {
	return &CronService{
		JanProvider: janProvider,
	}
}

func (cs *CronService) Start(ctx context.Context, ctab *crontab.Crontab) {
	cs.refreshJanModels(ctx)

	ctab.AddJob("* * * * *", func() {
		cs.refreshJanModels(ctx)
		environment_variables.EnvironmentVariables.LoadFromEnv()
	})
}

func (cs *CronService) refreshJanModels(ctx context.Context) {
	if cs == nil || cs.JanProvider == nil {
		return
	}

	if _, err := cs.JanProvider.RefreshModels(ctx); err != nil {
		logger.GetLogger().Warnf("cron service: failed to refresh Jan models: %v", err)
	}
}
