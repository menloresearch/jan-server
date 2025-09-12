package openai

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
