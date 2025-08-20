package userrepo

import (
	"context"
	"fmt"

	domain "menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"

	"gorm.io/gorm"
)

type UserGormRepository struct {
	query *gormgen.Query
	db    *gorm.DB
}

func NewUserGormRepository(db *gorm.DB) domain.UserRepository {
	return &UserGormRepository{
		query: gormgen.Use(db),
		db:    db,
	}
}

func (r *UserGormRepository) Create(ctx context.Context, u *domain.User) error {
	model := dbschema.NewSchemaUser(u)

	if err := r.query.User.WithContext(ctx).Create(model); err != nil {
		return err
	}

	u.ID = model.ID
	return nil
}

func (r *UserGormRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	model, err := r.query.User.WithContext(ctx).Where(r.query.User.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

func (r *UserGormRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	models, err := r.query.User.WithContext(ctx).Where(r.query.User.Email.Eq(email)).Find()
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, nil
	}

	if len(models) != 1 {
		return nil, fmt.Errorf("duplicated user email")
	}
	model := models[0]
	return model.EtoD(), nil
}
