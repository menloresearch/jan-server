package dbschema

import (
	"gorm.io/gorm"
	domain "menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(User{})
}

type User struct {
	BaseModel
	Name    string
	Email   string `gorm:"uniqueIndex"`
	Enabled bool
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

func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	domainApiKey, err := domain.NewApiKey(u.ID, "Default API Key for Jan Cloud", domain.ApiKeyServiceTypeJanCloud, nil)
	if err != nil {
		return err
	}
	return tx.Create(NewSchemaApiKey(domainApiKey)).Error
}
