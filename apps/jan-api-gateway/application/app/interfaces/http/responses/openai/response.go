package openai

import "context"

type ObjectKey string

const (
	ObjectKeyAdminApiKey ObjectKey = "organization.admin_api_key"
	ObjectKeyProject     ObjectKey = "organization.project"
)

type ApikeyType string

const (
	ApikeyTypeUser ApikeyType = "user"
)

type OwnerObject string

const (
	OwnerObjectOrganizationUser OwnerObject = "organization.user"
)

type OwnerRole string

const (
	OwnerRoleOwner OwnerObject = "owner"
)

// @Enum(list)
type ObjectTypeList string

const ObjectTypeListList ObjectTypeList = "list"

type ListResponse[T any] struct {
	Object  ObjectTypeList `json:"object"`
	Data    []T            `json:"data"`
	FirstID *string        `json:"first_id"`
	LastID  *string        `json:"last_id"`
	HasMore bool           `json:"has_more"`
	Total   int64          `json:"total"`
}

type NextFunc[T any] func(ctx context.Context, last T) ([]T, error)

func NewPaginator[T any](
	ctx context.Context,
	items []T,
	getID func(T) string,
	next NextFunc[T],
) (*ListResponse[T], error) {
	p := &ListResponse[T]{Data: items}
	if len(items) > 0 {
		firstID := getID(items[0])
		lastID := getID(items[len(items)-1])
		p.FirstID = &firstID
		p.LastID = &lastID
		more, err := next(ctx, items[len(items)-1])
		if err != nil {
			return nil, err
		}
		if len(more) > 0 {
			p.HasMore = true
		}
	}

	return p, nil
}
