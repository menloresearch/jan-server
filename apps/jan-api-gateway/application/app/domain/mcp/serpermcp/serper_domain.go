package serpermcp

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

type FetchWebpageRequest struct {
	Url string `json:"url"`
}

type FetchWebpageResponse struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
}
