package janinference

import (
	"bufio"
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	"menlo.ai/jan-api-gateway/app/utils/httpclients"
	"menlo.ai/jan-api-gateway/config/environment_variables"
	"resty.dev/v3"
)

// consider using "github.com/sashabaranov/go-openai"
var JanInferenceRestyClient *resty.Client

func Init() {
	JanInferenceRestyClient = httpclients.NewClient("JanInferenceClient")
	JanInferenceRestyClient.SetBaseURL(environment_variables.EnvironmentVariables.JAN_INFERENCE_MODEL_URL)
}

type JanInferenceClient struct {
	BaseURL string
}

func NewJanInferenceClient(ctx context.Context) *JanInferenceClient {
	return &JanInferenceClient{
		BaseURL: environment_variables.EnvironmentVariables.JAN_INFERENCE_MODEL_URL,
	}
}

func (client *JanInferenceClient) CreateChatCompletionStream(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) error {
	reqCtx, ok := ctx.(*gin.Context)
	if !ok {
		return fmt.Errorf("invalid context")
	}
	reqCtx.Writer.Header().Set("Content-Type", "text/event-stream")
	reqCtx.Writer.Header().Set("Cache-Control", "no-cache")
	reqCtx.Writer.Header().Set("Connection", "keep-alive")
	reqCtx.Writer.Header().Set("Transfer-Encoding", "chunked")

	req := JanInferenceRestyClient.R().SetBody(request)
	resp, err := req.
		SetDoNotParseResponse(true).
		Post("/v1/chat/completions")
	if err != nil {
		return err
	}
	defer resp.RawResponse.Body.Close()
	scanner := bufio.NewScanner(resp.RawResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()
		reqCtx.Writer.Write([]byte(line + "\n"))
		reqCtx.Writer.Flush()
	}
	reqCtx.Writer.Flush()
	return nil
}

// CreateChatCompletionStreamChunks returns chunks instead of writing to response
func (client *JanInferenceClient) CreateChatCompletionStreamChunks(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (<-chan string, error) {
	chunkChan := make(chan string, 100)

	go func() {
		defer close(chunkChan)

		req := JanInferenceRestyClient.R().SetBody(request)
		resp, err := req.
			SetDoNotParseResponse(true).
			Post("/v1/chat/completions")
		if err != nil {
			chunkChan <- fmt.Sprintf("error: %v", err)
			return
		}
		defer resp.RawResponse.Body.Close()

		scanner := bufio.NewScanner(resp.RawResponse.Body)
		for scanner.Scan() {
			line := scanner.Text()
			chunkChan <- line
		}
	}()

	return chunkChan, nil
}

// TODO: add timeout
func (client *JanInferenceClient) CreateChatCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	var chatCompletionResponse openai.ChatCompletionResponse
	_, err := JanInferenceRestyClient.R().
		SetContext(ctx).
		SetBody(request).
		SetResult(&chatCompletionResponse).
		SetHeader("Content-Type", "application/json").
		SetAuthToken(apiKey).
		Post("/v1/chat/completions")
	return &chatCompletionResponse, err
}

func (c *JanInferenceClient) GetModels(ctx context.Context) (*ModelsResponse, error) {
	var result ModelsResponse
	_, err := JanInferenceRestyClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetResult(&result).
		Get("/v1/models")
	return &result, err
}

type ModelsResponse struct {
	Object string                 `json:"object"`
	Data   []inferencemodel.Model `json:"data"`
}
