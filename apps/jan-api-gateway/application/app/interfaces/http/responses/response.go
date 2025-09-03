package responses

type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

type GeneralResponse[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type ListResponse[T any] struct {
	Status  string `json:"status"`
	Total   int64  `json:"total"`
	Results []T    `json:"results"`
}

const ResponseCodeOk = "000000"
