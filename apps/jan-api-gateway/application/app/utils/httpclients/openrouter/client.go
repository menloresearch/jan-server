package openrouter

import (
	"context"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/utils/httpclients"
	"menlo.ai/jan-api-gateway/config/environment_variables"
	"resty.dev/v3"
)

var RestyClient *resty.Client

func Init() {
	RestyClient = httpclients.NewClient("OpenRouterClient")
}

type Client struct {
	baseURL string
}

func NewClient() *Client {
	base := environment_variables.EnvironmentVariables.OPENROUTER_BASE_URL
	if base == "" {
		base = "https://openrouter.ai/api/v1"
	}
	return &Client{baseURL: base}
}

func (c *Client) CreateChatCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	var response openai.ChatCompletionResponse
	_, err := RestyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey)).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		SetResult(&response).
		Post(c.baseURL + "/chat/completions")
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) CreateChatCompletionStream(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	req := RestyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey)).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		SetDoNotParseResponse(true)

	resp, err := req.Post(c.baseURL + "/chat/completions")
	if err != nil {
		return nil, err
	}
	return resp.RawResponse.Body, nil
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
	Created int    `json:"created"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

func (c *Client) GetModels(ctx context.Context, apiKey string) (*ModelsResponse, error) {
	var response ModelsResponse
	_, err := RestyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey)).
		SetHeader("Content-Type", "application/json").
		SetResult(&response).
		Get(c.baseURL + "/models")
	if err != nil {
		return nil, err
	}
	return &response, nil
}
