package responses

type ErrorResponse struct {
	Code          string `json:"code"`
	Error         string `json:"error"`
	ErrorInstance error  `json:"-"`
}

type GeneralResponse[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type ListResponse[T any] struct {
	Status  string  `json:"status"`
	Total   int64   `json:"total"`
	Results []T     `json:"results"`
	FirstID *string `json:"first_id"`
	LastID  *string `json:"last_id"`
	HasMore bool    `json:"has_more"`
}

const ResponseCodeOk = "000000"
