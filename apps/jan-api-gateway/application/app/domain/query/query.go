package query

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	PageNumber int
	PageSize   int
}

func GetPaginationFromQuery(reqCtx *gin.Context) (*Pagination, error) {
	pageStr := reqCtx.Query("page")
	pageSizeStr := reqCtx.DefaultQuery("pageSize", "10")
	if pageStr == "" || pageSizeStr == "" {
		return &Pagination{
			PageNumber: 1,
			PageSize:   20,
		}, nil
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return nil, fmt.Errorf("invalid page number")
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		return nil, fmt.Errorf("invalid page size")
	}

	return &Pagination{
		PageNumber: page,
		PageSize:   pageSize,
	}, nil
}
