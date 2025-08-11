package responses

type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}
