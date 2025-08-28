package query

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Limit *int
	After *uint
	Order string
}

func GetPaginationFromQuery(reqCtx *gin.Context) (*Pagination, error) {
	limitStr := reqCtx.DefaultQuery("limit", "20")
	order := reqCtx.DefaultQuery("order", "asc")

	var limit *int

	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt < 1 {
			return nil, fmt.Errorf("invalid limit number")
		}
		limit = &limitInt
	}

	if order != "asc" && order != "desc" {
		return nil, fmt.Errorf("invalid order")
	}

	return &Pagination{
		Limit: limit,
		Order: order,
	}, nil
}
