package dbschema

import (
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(User{})
}

type User struct {
	BaseModel
	Name          string
	Email         string `gorm:"uniqueIndex"`
	Enabled       bool
	Organizations []OrganizationMember `gorm:"foreignKey:UserID"`
	Projects      []ProjectMember      `gorm:"foreignKey:UserID"`
}

func NewSchemaUser(u *user.User) *User {
	return &User{
		BaseModel: BaseModel{
			ID: u.ID,
		},
		Name:    u.Name,
		Email:   u.Email,
		Enabled: u.Enabled,
	}
}

func (u *User) EtoD() *user.User {
	return &user.User{
		ID:      u.ID,
		Name:    u.Name,
		Email:   u.Email,
		Enabled: u.Enabled,
	}
}
