package query

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Limit  *int
	Offset *int
	After  *uint
	Order  string
}

func GetPaginationFromQuery(reqCtx *gin.Context) (*Pagination, error) {
	limitStr := reqCtx.DefaultQuery("limit", "20")
	offsetStr := reqCtx.Query("offset")
	order := reqCtx.DefaultQuery("order", "asc")

	var limit *int
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt < 1 {
			return nil, fmt.Errorf("invalid limit number")
		}
		limit = &limitInt
	}

	var offset *int
	if offsetStr != "" {
		offsetInt, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, fmt.Errorf("invalid offset number")
		}
		offset = &offsetInt
	}

	if order != "asc" && order != "desc" {
		return nil, fmt.Errorf("invalid order")
	}

	return &Pagination{
		Limit:  limit,
		Offset: offset,
		Order:  order,
	}, nil
}
