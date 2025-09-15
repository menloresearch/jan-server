package responserepo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/response"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

// ResponseRepository implements the response repository interface
type ResponseRepository struct {
	db *gorm.DB
}

// NewResponseRepository creates a new response repository
func NewResponseRepository(db *gorm.DB) *ResponseRepository {
	return &ResponseRepository{db: db}
}

// Create creates a new response in the database
func (r *ResponseRepository) Create(ctx context.Context, resp *response.Response) error {
	// Generate public ID if not provided
	if resp.PublicID == "" {
		id, err := idgen.GenerateSecureID("resp", 42)
		if err != nil {
			return fmt.Errorf("failed to generate response ID: %w", err)
		}
		resp.PublicID = id
	}

	dbResponse := r.domainToDB(resp)
	if err := r.db.WithContext(ctx).Create(dbResponse).Error; err != nil {
		return fmt.Errorf("failed to create response: %w", err)
	}

	// Update the domain model with the generated ID
	resp.ID = dbResponse.ID
	resp.CreatedAt = dbResponse.CreatedAt
	resp.UpdatedAt = dbResponse.UpdatedAt

	return nil
}

// Update updates an existing response in the database
func (r *ResponseRepository) Update(ctx context.Context, resp *response.Response) error {
	dbResponse := r.domainToDB(resp)
	if err := r.db.WithContext(ctx).Save(dbResponse).Error; err != nil {
		return fmt.Errorf("failed to update response: %w", err)
	}

	resp.UpdatedAt = dbResponse.UpdatedAt
	return nil
}

// DeleteByID deletes a response by ID
func (r *ResponseRepository) DeleteByID(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&dbschema.Response{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete response: %w", err)
	}
	return nil
}

// FindByID finds a response by ID
func (r *ResponseRepository) FindByID(ctx context.Context, id uint) (*response.Response, error) {
	var dbResponse dbschema.Response
	if err := r.db.WithContext(ctx).First(&dbResponse, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find response by ID: %w", err)
	}

	return r.dbToDomain(&dbResponse), nil
}

// FindByPublicID finds a response by public ID
func (r *ResponseRepository) FindByPublicID(ctx context.Context, publicID string) (*response.Response, error) {
	var dbResponse dbschema.Response
	if err := r.db.WithContext(ctx).Where("public_id = ?", publicID).First(&dbResponse).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find response by public ID: %w", err)
	}

	return r.dbToDomain(&dbResponse), nil
}

// FindByFilter finds responses by filter criteria
func (r *ResponseRepository) FindByFilter(ctx context.Context, filter response.ResponseFilter, pagination *query.Pagination) ([]*response.Response, error) {
	var dbResponses []dbschema.Response
	query := r.db.WithContext(ctx).Model(&dbschema.Response{})

	// Apply filters
	if filter.PublicID != nil {
		query = query.Where("public_id = ?", *filter.PublicID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.ConversationID != nil {
		query = query.Where("conversation_id = ?", *filter.ConversationID)
	}
	if filter.Model != nil {
		query = query.Where("model = ?", *filter.Model)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", string(*filter.Status))
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filter.CreatedBefore)
	}

	// Apply pagination
	if pagination != nil {
		if pagination.Limit != nil && *pagination.Limit > 0 {
			query = query.Limit(*pagination.Limit)
		}
		if pagination.Offset != nil && *pagination.Offset > 0 {
			query = query.Offset(*pagination.Offset)
		}
	}

	// Order by created_at desc
	query = query.Order("created_at DESC")

	if err := query.Find(&dbResponses).Error; err != nil {
		return nil, fmt.Errorf("failed to find responses by filter: %w", err)
	}

	responses := make([]*response.Response, len(dbResponses))
	for i, dbResp := range dbResponses {
		responses[i] = r.dbToDomain(&dbResp)
	}

	return responses, nil
}

// Count counts responses by filter criteria
func (r *ResponseRepository) Count(ctx context.Context, filter response.ResponseFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&dbschema.Response{})

	// Apply filters
	if filter.PublicID != nil {
		query = query.Where("public_id = ?", *filter.PublicID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.ConversationID != nil {
		query = query.Where("conversation_id = ?", *filter.ConversationID)
	}
	if filter.Model != nil {
		query = query.Where("model = ?", *filter.Model)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", string(*filter.Status))
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filter.CreatedBefore)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count responses: %w", err)
	}

	return count, nil
}

// FindByUserID finds responses by user ID
func (r *ResponseRepository) FindByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*response.Response, error) {
	filter := response.ResponseFilter{UserID: &userID}
	return r.FindByFilter(ctx, filter, pagination)
}

// FindByConversationID finds responses by conversation ID
func (r *ResponseRepository) FindByConversationID(ctx context.Context, conversationID uint, pagination *query.Pagination) ([]*response.Response, error) {
	filter := response.ResponseFilter{ConversationID: &conversationID}
	return r.FindByFilter(ctx, filter, pagination)
}

// domainToDB converts domain model to database model
func (r *ResponseRepository) domainToDB(resp *response.Response) *dbschema.Response {
	return dbschema.NewSchemaResponse(resp)
}

// dbToDomain converts database model to domain model
func (r *ResponseRepository) dbToDomain(dbResp *dbschema.Response) *response.Response {
	return dbResp.EtoD()
}
