package userrepo

import (
	"context"
	"fmt"

	domain "menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
)

type UserGormRepository struct {
	db *transaction.Database
}

func NewUserGormRepository(db *transaction.Database) domain.UserRepository {
	return &UserGormRepository{
		db: db,
	}
}

func (r *UserGormRepository) Create(ctx context.Context, u *domain.User) error {
	model := dbschema.NewSchemaUser(u)
	if err := r.db.GetQuery(ctx).User.WithContext(ctx).Create(model); err != nil {
		return err
	}
	u.ID = model.ID
	return nil
}

func (r *UserGormRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	query := r.db.GetQuery(ctx)
	model, err := query.User.WithContext(ctx).Where(query.User.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

func (r *UserGormRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := r.db.GetQuery(ctx)
	models, err := query.User.WithContext(ctx).Where(query.User.Email.Eq(email)).Find()
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
