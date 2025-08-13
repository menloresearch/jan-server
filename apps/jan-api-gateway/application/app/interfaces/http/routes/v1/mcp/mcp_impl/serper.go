package mcpimpl

import (
	"context"
	"encoding/json"

	mcp_golang "github.com/metoro-io/mcp-golang"
	mcpgolang "github.com/metoro-io/mcp-golang"
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
	Q           string  `json:"q" jsonschema:"required,description=Search query"`
	GL          *string `json:"gl,omitempty" jsonschema:"description=Country code, e.g. us, sg"`
	HL          *string `json:"hl,omitempty" jsonschema:"description=Language code, e.g. en, zh-TW"`
	Location    *string `json:"location,omitempty" jsonschema:"description=Location"`
	Num         *int    `json:"num,omitempty" jsonschema:"description=Number of results"`
	Page        *int    `json:"page,omitempty" jsonschema:"description=Page number"`
	Autocorrect *bool   `json:"autocorrect,omitempty" jsonschema:"description=Enable autocorrect"`
	Tbs         *string `json:"tbs,omitempty" jsonschema:"description=Time filter, e.g. qdr:h, qdr:d, qdr:w, qdr:m, qdr:y"`
}

type SerperScrapeArgs struct {
	Url string `json:"url" jsonschema:"required,description=The full URL of the page to scrape"`
}

func (s *SerperMCP) RegisterTool(handler *mcpgolang.Server) {
	handler.RegisterTool(
		"websearch",
		"Search Google results",
		func(args SerperSearchArgs) (*mcp_golang.ToolResponse, error) {
			req := serpermcp.SearchRequest{
				Q:           args.Q,
				GL:          ptr.ToString("us"),
				Num:         ptr.ToInt(10),
				Page:        ptr.ToInt(1),
				Autocorrect: ptr.ToBool(true),
			}
			if args.GL != nil {
				req.GL = args.GL
			}
			if args.HL != nil {
				req.HL = args.HL
			}
			if args.Location != nil {
				req.Location = args.Location
			}
			if args.Num != nil {
				req.Num = args.Num
			}
			if args.Page != nil {
				req.Page = args.Page
			}
			if args.Autocorrect != nil {
				req.Autocorrect = args.Autocorrect
			}
			if args.Tbs != nil {
				tbs := serpermcp.TBSTimeRange(*args.Tbs)
				req.TBS = &tbs
			}
			searchResp, err := s.SerperService.Search(context.Background(), req)
			if err != nil {
				return nil, err
			}
			jsonBytes, err := json.Marshal(searchResp)
			if err != nil {
				return nil, err
			}

			return mcp_golang.NewToolResponse(
				mcp_golang.NewTextContent(string(jsonBytes)),
			), nil
		},
	)

	handler.RegisterTool(
		"webscrape",
		"Scrape and return structured web content from a given URL",
		func(args SerperScrapeArgs) (*mcp_golang.ToolResponse, error) {
			req := serpermcp.FetchWebpageRequest{
				Url: args.Url,
			}
			scrapeResp, err := s.SerperService.FetchWebpage(context.Background(), req)
			if err != nil {
				return nil, err
			}
			jsonBytes, err := json.Marshal(scrapeResp)
			if err != nil {
				return nil, err
			}

			return mcp_golang.NewToolResponse(
				mcp_golang.NewTextContent(string(jsonBytes)),
			), nil
		},
	)
}
