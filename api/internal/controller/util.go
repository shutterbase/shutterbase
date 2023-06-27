package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shutterbase/shutterbase/internal/repository"
)

func getPaginationParameters(c *gin.Context) repository.PaginationParameters {
	limitParameter := c.DefaultQuery("limit", "100")
	offsetParameter := c.DefaultQuery("offset", "0")
	searchParameter := c.DefaultQuery("search", "")
	sortParameter := c.DefaultQuery("sort", "")
	orderDirectionParameter := c.DefaultQuery("order", "")

	limit, err := strconv.Atoi(limitParameter)
	if err != nil {
		limit = 100
	}
	offset, err := strconv.Atoi(offsetParameter)
	if err != nil {
		offset = 0
	}

	return repository.PaginationParameters{
		Limit:          limit,
		Offset:         offset,
		Search:         searchParameter,
		Sort:           sortParameter,
		OrderDirection: orderDirectionParameter,
	}
}
