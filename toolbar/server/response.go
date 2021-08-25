package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// response describes the response standard. Code & Msg are always present.
// Data is present for a success response only.
type response struct {
	Code   int                    `json:"code"`
	Msg    string                 `json:"msg"`
	Result map[string]interface{} `json:"result,omitempty"`
}

func respondErrorResp(c *gin.Context, err error) {
	log.WithFields(log.Fields{
		"url":     c.Request.URL,
		"request": c.Value(reqBodyLabel),
		"err":     err,
	}).Error("request fail")
	resp := formatErrResp(err)
	c.AbortWithStatusJSON(http.StatusOK, resp)
}

func respondSuccessResp(c *gin.Context, data interface{}) {
	result := make(map[string]interface{})
	result["data"] = data
	c.AbortWithStatusJSON(http.StatusOK, response{Code: 200, Result: result})
}

type links struct {
	Next string `json:"next,omitempty"`
}

func respondSuccessPaginationResp(c *gin.Context, data interface{}, paginationInfo *PaginationInfo) {
	url := fmt.Sprintf("%v", c.Request.URL)
	base := strings.Split(url, "?")[0]
	start := paginationInfo.Start
	limit := paginationInfo.Limit

	l := links{}
	if paginationInfo.HasNext {
		// To efficiently build a string using Write methods
		// https://stackoverflow.com/questions/1760757/how-to-efficiently-concatenate-strings-in-go
		// https://tip.golang.org/pkg/strings/#Builder
		var b strings.Builder
		fmt.Fprintf(&b, "%s?limit=%d&start=%d", base, limit, start+limit)
		l.Next = b.String()
	}
	result := make(map[string]interface{})
	result["data"] = data
	result["start"] = start
	result["limit"] = limit
	result["_links"] = l

	c.AbortWithStatusJSON(http.StatusOK, response{
		Code:   http.StatusOK,
		Result: result,
	})
}
