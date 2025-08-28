package openai

type ObjectKey string

const (
	OrganizationAdminApiKey ObjectKey = "organization.admin_api_key"
)

type OwnerType string

const (
	OwnerTypeUser OwnerType = "user"
)

type OwnerObject string

const (
	OwnerObjectOrganizationUser OwnerObject = "organization.user"
)

type OwnerRole string

const (
	OwnerRoleOwner OwnerObject = "owner"
)
