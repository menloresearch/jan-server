package mcpimpl

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	mcpservice "menlo.ai/jan-api-gateway/app/domain/mcp"
	"menlo.ai/jan-api-gateway/app/domain/mcp/serpermcp"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type SerperMCP struct {
	SerperService *serpermcp.SerperService
}

func NewSerperMCP(serperService *serpermcp.SerperService) *SerperMCP {
	return &SerperMCP{
		SerperService: serperService,
	}
}

type SerperSearchArgs struct {
	Q           string  `json:"q" jsonschema:"required,description=Search query string"`
	GL          *string `json:"gl,omitempty" jsonschema:"description=Optional region code for search results in ISO 3166-1 alpha-2 format (e.g., 'us')"`
	HL          *string `json:"hl,omitempty" jsonschema:"description=Optional language code for search results in ISO 639-1 format (e.g., 'en')"`
	Location    *string `json:"location,omitempty" jsonschema:"description=Optional location for search results (e.g., 'SoHo, New York, United States', 'California, United States')"`
	Num         *int    `json:"num,omitempty" jsonschema:"description=Number of results to return (default: 10)"`
	Tbs         *string `json:"tbs,omitempty" jsonschema:"description=Time-based search filter ('qdr:h' for past hour, 'qdr:d' for past day, 'qdr:w' for past week, 'qdr:m' for past month, 'qdr:y' for past year)"`
	Page        *int    `json:"page,omitempty" jsonschema:"description=Page number of results to return (default: 1)"`
	Autocorrect *bool   `json:"autocorrect,omitempty" jsonschema:"description=Whether to autocorrect spelling in query"`
}

type SerperScrapeArgs struct {
	Url             string `json:"url" jsonschema:"required,description=The URL of webpage to scrape"`
	IncludeMarkdown *bool  `json:"includeMarkdown,omitempty" jsonschema:"description=Whether to include markdown content"`
}

func (s *SerperMCP) RegisterTool(handler *mcpserver.MCPServer) {
	handler.AddTool(
		mcp.NewTool("google_search",
			mcpservice.ReflectToMCPOptions(
				"Tool to perform web searches via Serper API and retrieve rich results. It is able to retrieve organic search results, people also ask, related searches, and knowledge graph.",
				SerperSearchArgs{},
			)...,
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			q, err := req.RequireString("q")
			if err != nil {
				return nil, err
			}
			searchReq := serpermcp.SearchRequest{
				Q:           q,
				GL:          ptr.ToString(req.GetString("gl", "us")),
				Num:         ptr.ToInt(req.GetInt("num", 10)),
				Page:        ptr.ToInt(req.GetInt("page", 1)),
				Autocorrect: ptr.ToBool(req.GetBool("autocorrect", true)),
			}
			hl := req.GetString("hl", "")
			if hl != "" {
				searchReq.HL = &hl
			}
			location := req.GetString("location", "")
			if location != "" {
				searchReq.Location = &location
			}

			searchResp, err := s.SerperService.Search(ctx, searchReq)
			if err != nil {
				return nil, err
			}
			jsonBytes, err := json.Marshal(searchResp)
			if err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)
	handler.AddTool(
		mcp.NewTool("scrape",
			mcpservice.ReflectToMCPOptions(
				"This is a tool to scrape a webpage and retrieve the text, with an option to provide the output in Markdown format.",
				SerperScrapeArgs{},
			)...,
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			url, err := req.RequireString("url")
			if err != nil {
				return nil, err
			}
			scrapeReq := serpermcp.FetchWebpageRequest{
				Url:             url,
				IncludeMarkdown: ptr.ToBool(req.GetBool("includeMarkdown", false)),
			}
			searchResp, err := s.SerperService.FetchWebpage(ctx, scrapeReq)
			if err != nil {
				return nil, err
			}
			jsonBytes, err := json.Marshal(searchResp)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)
}
