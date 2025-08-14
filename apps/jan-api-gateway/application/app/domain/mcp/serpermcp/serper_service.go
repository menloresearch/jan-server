package serpermcp

import (
	"context"

	"menlo.ai/jan-api-gateway/app/utils/httpclients/serper"
)

type SerperService struct {
	SerperClient *serper.SerperClient
}

func NewSerperService() *SerperService {
	return &SerperService{
		SerperClient: serper.NewSerperClient(),
	}
}

func (s *SerperService) Search(ctx context.Context, query SearchRequest) (*SearchResponse, error) {
	var tbs *serper.TBSTimeRange
	request := serper.SearchRequest{
		Q:           query.Q,
		GL:          query.GL,
		HL:          query.HL,
		Location:    query.Location,
		Num:         query.Num,
		Page:        query.Page,
		Autocorrect: query.Autocorrect,
		TBS:         tbs,
	}
	resp, err := s.SerperClient.Search(ctx, request)
	if err != nil {
		return nil, err
	}

	return &SearchResponse{
		SearchParameters: resp.SearchParameters,
		Organic:          resp.Organic,
		KnowledgeGraph:   resp.KnowledgeGraph,
		Images:           resp.Images,
		News:             resp.News,
		AnswerBox:        resp.AnswerBox,
	}, nil
}

func (s *SerperService) FetchWebpage(ctx context.Context, query FetchWebpageRequest) (*FetchWebpageResponse, error) {
	request := serper.FetchWebpageRequest{
		Url:             query.Url,
		IncludeMarkdown: query.IncludeMarkdown,
	}
	resp, err := s.SerperClient.FetchWebpage(ctx, request)
	if err != nil {
		return nil, err
	}

	return &FetchWebpageResponse{
		Text:     resp.Text,
		Metadata: resp.Metadata,
	}, nil
}
