package gemini

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/utils/httpclients"
	"menlo.ai/jan-api-gateway/config/environment_variables"
	"resty.dev/v3"
)

var RestyClient *resty.Client

func Init() {
	RestyClient = httpclients.NewClient("GeminiClient")
}

type Client struct {
	baseURL string
}

func NewClient() *Client {
	base := environment_variables.EnvironmentVariables.GEMINI_BASE_URL
	if base == "" {
		base = "https://generativelanguage.googleapis.com/v1beta/openai"
	}
	return &Client{baseURL: base}
}

type modelsAPIResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID          string `json:"id"`
		Object      string `json:"object"`
		OwnedBy     string `json:"owned_by"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
}

type Model struct {
	ID      string
	Object  string
	Created int
	OwnedBy string
}

type ModelsResponse struct {
	Object string
	Data   []Model
}

func (c *Client) CreateChatCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	var resp openai.ChatCompletionResponse
	_, err := RestyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey)).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		SetResult(&resp).
		Post(c.baseURL + "/chat/completions")
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CreateChatCompletionStream(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		reqBody := request
		reqBody.Stream = true // <— KEY

		req := RestyClient.R().SetBody(reqBody)

		resp, err := req.
			SetContext(ctx).
			SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey)).
			SetHeader("Content-Type", "application/json").
			SetDoNotParseResponse(true).
			Post(c.baseURL + "/chat/completions")
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		defer resp.RawResponse.Body.Close()

		if resp.IsError() {
			body, readErr := io.ReadAll(resp.RawResponse.Body)
			if readErr != nil {
				writer.CloseWithError(fmt.Errorf("gemini provider: streaming request failed with status %d", resp.StatusCode()))
				return
			}
			writer.CloseWithError(fmt.Errorf("gemini provider: streaming request failed with status %d: %s", resp.StatusCode(), strings.TrimSpace(string(body))))
			return
		}

		// Pipe the SSE bytes through; caller should parse "data: ..." lines.
		if _, err = io.Copy(writer, resp.RawResponse.Body); err != nil {
			writer.CloseWithError(err)
		}
	}()

	return reader, nil
}

func (c *Client) GetModels(ctx context.Context, apiKey string) (*ModelsResponse, error) {
	var resp modelsAPIResponse
	_, err := RestyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey)).
		SetHeader("Content-Type", "application/json").
		SetResult(&resp).
		Get(c.baseURL + "/models")
	if err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(resp.Data))
	now := int(time.Now().Unix())
	for _, model := range resp.Data {
		models = append(models, Model{
			ID:      model.ID,
			Object:  model.Object,
			Created: now,
			OwnedBy: model.OwnedBy,
		})
	}
	return &ModelsResponse{Object: resp.Object, Data: models}, nil
}
