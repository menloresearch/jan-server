package gemini

import (
	"context"
	"fmt"
	"io"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/inference"
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
		base = "https://generativelanguage.googleapis.com/v1beta"
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
	Models []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
	} `json:"models"`
}

func (c *Client) CreateChatCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	geminiReq := convertToGeminiRequest(request)
	var geminiResp generateContentResponse
	endpoint := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, request.Model)
	_, err := RestyClient.R().
		SetContext(ctx).
		SetQueryParam("key", apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(geminiReq).
		SetResult(&geminiResp).
		Post(endpoint)
	if err != nil {
		return nil, err
	}
	return convertToOpenAIResponse(request.Model, geminiResp), nil
}

func (c *Client) CreateChatCompletionStream(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	return nil, fmt.Errorf("gemini streaming completions not supported")
}

func (c *Client) GetModels(ctx context.Context, apiKey string) (*inference.ModelsResponse, error) {
	var resp modelsResponse
	_, err := RestyClient.R().
		SetContext(ctx).
		SetQueryParam("key", apiKey).
		SetHeader("Content-Type", "application/json").
		SetResult(&resp).
		Get(c.baseURL + "/models")
	if err != nil {
		return nil, err
	}

	models := make([]inference.Model, 0, len(resp.Models))
	now := int(time.Now().Unix())
	for _, model := range resp.Models {
		models = append(models, inference.Model{
			ID:      model.Name,
			Object:  "model",
			Created: now,
			OwnedBy: "google",
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
