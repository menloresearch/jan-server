package responses

type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

type GeneralResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data"`
}
