package dbschema

import (
	"time"

	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(Project{})
	database.RegisterSchemaForAutoMigrate(ProjectMember{})
}

type Project struct {
	BaseModel
	Name           string          `gorm:"size:128;not null"`
	PublicID       string          `gorm:"type:varchar(50);uniqueIndex;not null"`
	Status         string          `gorm:"type:varchar(20);not null;default:'active';index"`
	OrganizationID uint            `gorm:"not null;index"`
	ArchivedAt     *time.Time      `gorm:"column:archived_at;index"`
	Members        []ProjectMember `gorm:"foreignKey:ProjectID"`
}

type ProjectMemberRole string

const (
	ProjectMemberRoleOwner  ProjectMemberRole = "owner"
	ProjectMemberRoleMember ProjectMemberRole = "member"
)

type ProjectMember struct {
	BaseModel
	UserID    uint   `gorm:"not null;index:idx_user_proj,unique"`
	ProjectID uint   `gorm:"not null;index:idx_user_proj,unique"`
	Role      string `gorm:"type:varchar(20);not null"`
}

func NewSchemaProject(p *project.Project) *Project {
	return &Project{
		BaseModel: BaseModel{
			ID: p.ID,
		},
		Name:           p.Name,
		PublicID:       p.PublicID,
		Status:         p.Status,
		ArchivedAt:     p.ArchivedAt,
		OrganizationID: p.OrganizationID,
	}
}

func (p *Project) EtoD() *project.Project {
	return &project.Project{
		ID:             p.ID,
		Name:           p.Name,
		PublicID:       p.PublicID,
		ArchivedAt:     p.ArchivedAt,
		Status:         p.Status,
		OrganizationID: p.OrganizationID,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func NewSchemaProjectMember(p *project.ProjectMember) *ProjectMember {
	return &ProjectMember{
		BaseModel: BaseModel{
			ID: p.ID,
		},
		UserID:    p.UserID,
		ProjectID: p.ProjectID,
		Role:      p.Role,
	}
}
