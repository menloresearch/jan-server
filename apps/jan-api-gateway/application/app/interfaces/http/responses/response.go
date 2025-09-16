package responses

type ErrorResponse struct {
	Code          string `json:"code"`
	Error         string `json:"error"`
	ErrorInstance error  `json:"-"`
}

type GeneralResponse[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type ListResponse[T any] struct {
	Status  string  `json:"status"`
	Total   int64   `json:"total"`
	Results []T     `json:"results"`
	FirstID *string `json:"first_id"`
	LastID  *string `json:"last_id"`
	HasMore bool    `json:"has_more"`
}

const ResponseCodeOk = "000000"

type PageCursor struct {
	FirstID *string
	LastID  *string
	HasMore bool
	Total   int64
}

func BuildCursorPage[T any](
	items []*T,
	getID func(*T) *string,
	hasMoreFunc func() ([]*T, error),
	CountFunc func() (int64, error),
) (*PageCursor, error) {
	cursorPage := &PageCursor{}
	if len(items) > 0 {
		cursorPage.FirstID = getID(items[0])
		cursorPage.LastID = getID(items[len(items)-1])
		moreRecords, err := hasMoreFunc()
		if len(moreRecords) > 0 {
			cursorPage.HasMore = true
		}
		if err != nil {
			return nil, err
		}
	}
	count, err := CountFunc()
	if err != nil {
		return cursorPage, err
	}
	cursorPage.Total = count
	return cursorPage, nil
}
