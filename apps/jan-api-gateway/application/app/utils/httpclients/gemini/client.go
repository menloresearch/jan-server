package gemini

import (
	"context"
	"fmt"
	"io"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/inference"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
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

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type generateContentRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content struct {
			Role  string       `json:"role"`
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

type modelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID          string `json:"id"`
		Object      string `json:"object"`
		OwnedBy     string `json:"owned_by"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
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
	resp, err := RestyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey)).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		SetDoNotParseResponse(true).
		Post(c.baseURL + "/chat/completions")
	if err != nil {
		return nil, err
	}
	return resp.RawResponse.Body, nil
}

func (c *Client) GetModels(ctx context.Context, apiKey string) (*inference.ModelsResponse, error) {
	var resp modelsResponse
	_, err := RestyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey)).
		SetHeader("Content-Type", "application/json").
		SetResult(&resp).
		Get(c.baseURL + "/models")
	if err != nil {
		return nil, err
	}

	models := make([]inference.InferenceProviderModel, 0, len(resp.Data))
	now := int(time.Now().Unix())
	for _, model := range resp.Data {
		models = append(models, inference.InferenceProviderModel{
			Model: inferencemodel.Model{
				ID:      model.ID,
				Object:  model.Object,
				Created: now,
				OwnedBy: model.OwnedBy,
			},
		})
	}
	return &inference.ModelsResponse{Object: "list", Data: models}, nil
}

func convertToGeminiRequest(request openai.ChatCompletionRequest) generateContentRequest {
	var systemContent *geminiContent
	contents := make([]geminiContent, 0, len(request.Messages))
	for _, msg := range request.Messages {
		part := geminiPart{Text: msg.Content}
		content := geminiContent{Role: msg.Role, Parts: []geminiPart{part}}
		if msg.Role == "system" {
			if systemContent == nil {
				systemContent = &geminiContent{Parts: []geminiPart{part}}
			} else {
				systemContent.Parts = append(systemContent.Parts, part)
			}
			continue
		}
		contents = append(contents, content)
	}
	return generateContentRequest{
		Contents:          contents,
		SystemInstruction: systemContent,
	}
}

func convertToOpenAIResponse(model string, geminiResp generateContentResponse) *openai.ChatCompletionResponse {
	response := &openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
	}
	if len(geminiResp.Candidates) == 0 {
		return response
	}
	candidate := geminiResp.Candidates[0]
	var text string
	if len(candidate.Content.Parts) > 0 {
		text = candidate.Content.Parts[0].Text
	}
	choice := openai.ChatCompletionChoice{
		Index: 0,
		Message: openai.ChatCompletionMessage{
			Role:    candidate.Content.Role,
			Content: text,
		},
		FinishReason: openai.FinishReason(candidate.FinishReason),
	}
	response.Choices = []openai.ChatCompletionChoice{choice}
	return response
}
