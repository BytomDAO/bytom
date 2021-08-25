package server

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bytom/bytom/errors"
)

const (
	defaultSatrtStr = "0"
	defaultLimitStr = "10"
	maxPageLimit    = 1000
)

var (
	errParsePaginationStart = fmt.Errorf("parse pagination start")
	errParsePaginationLimit = fmt.Errorf("parse pagination limit")
)

type PaginationQuery struct {
	Start uint64 `json:"start"`
	Limit uint64 `json:"limit"`
}

// parsePagination request meets the standard on https://developer.atlassian.com/server/confluence/pagination-in-the-rest-api/
func parsePagination(c *gin.Context) (*PaginationQuery, error) {
	startStr := c.DefaultQuery("start", defaultSatrtStr)
	limitStr := c.DefaultQuery("limit", defaultLimitStr)

	start, err := strconv.ParseUint(startStr, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, errParsePaginationStart)
	}

	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, errParsePaginationLimit)
	}

	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	return &PaginationQuery{
		Start: start,
		Limit: limit,
	}, nil
}

type PaginationInfo struct {
	Start   uint64
	Limit   uint64
	HasNext bool
}

func processPaginationIfPresent(fun handlerFun, args []interface{}, result []interface{}, context *gin.Context) bool {
	ft := reflect.TypeOf(fun)
	if ft.NumIn() != 3 {
		return false
	}

	list := result[0]
	size := reflect.ValueOf(list).Len()
	query := args[2].(*PaginationQuery)

	paginationInfo := &PaginationInfo{Start: query.Start, Limit: query.Limit, HasNext: size == int(query.Limit)}
	respondSuccessPaginationResp(context, list, paginationInfo)
	return true
}
