package jan

type JanListResponse[T any] struct {
	Status  string  `json:"status"`
	Total   int64   `json:"total"`
	Results []T     `json:"results"`
	FirstID *string `json:"first_id"`
	LastID  *string `json:"last_id"`
	HasMore bool    `json:"has_more"`
}
