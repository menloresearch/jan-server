package dbschema

import (
	"menlo.ai/jan-api-gateway/app/domain/workspace"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(Workspace{})
}

type Workspace struct {
	BaseModel
	PublicID      string         `gorm:"type:varchar(50);uniqueIndex;not null"`
	UserID        uint           `gorm:"not null;index"`
	Name          string         `gorm:"type:varchar(255);not null"`
	Instruction   *string        `gorm:"type:text"`
	Conversations []Conversation `gorm:"foreignKey:WorkspacePublicID;references:PublicID;constraint:OnDelete:CASCADE;"`
	User          User           `gorm:"foreignKey:UserID"`
}

func NewSchemaWorkspace(w *workspace.Workspace) *Workspace {
	return &Workspace{
		BaseModel:   BaseModel{ID: w.ID},
		PublicID:    w.PublicID,
		UserID:      w.UserID,
		Name:        w.Name,
		Instruction: w.Instruction,
	}
}

func (w *Workspace) EtoD() *workspace.Workspace {
	var instruction *string
	if w.Instruction != nil {
		value := *w.Instruction
		instruction = &value
	}

	return &workspace.Workspace{
		ID:          w.ID,
		PublicID:    w.PublicID,
		UserID:      w.UserID,
		Name:        w.Name,
		Instruction: instruction,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}
