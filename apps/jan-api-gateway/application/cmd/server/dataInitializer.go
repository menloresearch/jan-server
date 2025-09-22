package main

import "menlo.ai/jan-api-gateway/app/domain/auth"

type DataInitializer struct {
	authService *auth.AuthService
}

func (*DataInitializer) Install() error {
	return nil
}

func (d *DataInitializer) installDefaultOrganization() error {
	d.authService.o
	return nil
}
