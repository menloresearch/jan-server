package serper

import (
	"context"
	"fmt"

	"menlo.ai/jan-api-gateway/app/utils/httpclients"
	"menlo.ai/jan-api-gateway/config/environment_variables"
	"resty.dev/v3"
)

var SerperRestyClient *resty.Client

func Init() {
	SerperRestyClient = httpclients.NewClient("SerperClient")
}

type SerperClient struct {
	apiKey string
}

func NewSerperClient() *SerperClient {
	return &SerperClient{
		apiKey: environment_variables.EnvironmentVariables.SERPER_API_KEY,
	}
}

type TBSTimeRange string

const (
	TBSAny       TBSTimeRange = ""
	TBSPastHour  TBSTimeRange = "qdr:h"
	TBSPastDay   TBSTimeRange = "qdr:d"
	TBSPastWeek  TBSTimeRange = "qdr:w"
	TBSPastMonth TBSTimeRange = "qdr:m"
	TBSPastYear  TBSTimeRange = "qdr:y"
)

type SearchRequest struct {
	Q           string        `json:"q"`
	GL          *string       `json:"gl,omitempty"`
	HL          *string       `json:"hl,omitempty"`
	Location    *string       `json:"location,omitempty"`
	Num         *int          `json:"num,omitempty"`
	Page        *int          `json:"page,omitempty"`
	Autocorrect *bool         `json:"autocorrect,omitempty"`
	TBS         *TBSTimeRange `json:"tbs,omitempty"`
}

type SearchResponse struct {
	SearchParameters map[string]interface{}   `json:"searchParameters"`
	Organic          []map[string]interface{} `json:"organic"`
	KnowledgeGraph   map[string]interface{}   `json:"knowledgeGraph,omitempty"`
	Images           []map[string]interface{} `json:"images,omitempty"`
	News             []map[string]interface{} `json:"news,omitempty"`
	AnswerBox        map[string]interface{}   `json:"answerBox,omitempty"`
}

// TODO: add timeout
func (c *SerperClient) Search(ctx context.Context, query SearchRequest) (*SearchResponse, error) {
	var result SearchResponse

	resp, err := SerperRestyClient.R().
		SetContext(ctx).
		SetHeader("X-API-KEY", c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(query).
		SetResult(&result).
		Post("https://google.serper.dev/search")

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("serper API error: %s", resp.Status())
	}

	return &result, nil
}

type FetchWebpageRequest struct {
	Url             string `json:"url"`
	IncludeMarkdown *bool  `json:"includeMarkdown"`
}

type FetchWebpageResponse struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
}

// TODO: add timeout
func (c *SerperClient) FetchWebpage(ctx context.Context, query FetchWebpageRequest) (*FetchWebpageResponse, error) {
	var result FetchWebpageResponse
	resp, err := SerperRestyClient.R().
		SetContext(ctx).
		SetHeader("X-API-KEY", c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(query).
		SetResult(&result).
		Post("https://scrape.serper.dev")

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("serper API error: %s", resp.Status())
	}
	return &result, nil
}
