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
