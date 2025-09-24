package main

import (
	"context"

	"menlo.ai/jan-api-gateway/app/domain/auth"
)

type DataInitializer struct {
	authService *auth.AuthService
}

func (d *DataInitializer) Install(ctx context.Context) error {
	err := d.installDefaultOrganization(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataInitializer) installDefaultOrganization(ctx context.Context) error {
	return d.authService.InitOrganization(ctx)
}
