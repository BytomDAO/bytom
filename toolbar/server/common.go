package server

import (
	"reflect"

	"github.com/gin-gonic/gin"
)

const (
	serverLabel  = "server_label"
	reqBodyLabel = "request_body_label"
)

var (
	errorType           = reflect.TypeOf((*error)(nil)).Elem()
	contextType         = reflect.TypeOf((*gin.Context)(nil))
	paginationQueryType = reflect.TypeOf((*PaginationQuery)(nil))
)
