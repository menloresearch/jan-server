package janinference

import (
	"context"
	"fmt"
	"time"

	"menlo.ai/jan-api-gateway/app/utils/httpclients"
	"menlo.ai/jan-api-gateway/config/environment_variables"
	"resty.dev/v3"
)

// consider using "github.com/sashabaranov/go-openai"
var JanInferenceRestyClient *resty.Client

func Init() {
	JanInferenceRestyClient = resty.NewWithClient(httpclients.RestyClient.Client())
	JanInferenceRestyClient.SetBaseURL(environment_variables.EnvironmentVariables.JAN_INFERENCE_MODEL_URL)
}

type JanInferenceClient struct {
}

func NewJanInferenceClient() *JanInferenceClient {
	return &JanInferenceClient{}
}

func (client *JanInferenceClient) CreateChatCompletion(ctx context.Context, apiKey string, request ChatCompletionRequest) (*ChatCompletionResponse, error) {
	var chatCompletionResponse ChatCompletionResponse
	resp, err := JanInferenceRestyClient.R().
		SetContext(ctx).
		SetBody(request).
		SetResult(&chatCompletionResponse).
		SetHeader("Content-Type", "application/json").
		SetAuthToken(apiKey).
		SetTimeout(30 * time.Second).
		Post("/v1/chat/completions")
	fmt.Print(resp)
	return &chatCompletionResponse, err
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type ChoiceMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int           `json:"index"`
	Message      ChoiceMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}
