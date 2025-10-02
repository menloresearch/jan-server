package contextkeys

type RequestId struct{}
type HttpClientStartsAt struct{}
type HttpClientRequestBody struct{}
type TransactionContextKey struct{}

const SkipMiddleware = "SkipMiddleware"
const OrganizationID = "organization_id"
const UserID = "user_id"
