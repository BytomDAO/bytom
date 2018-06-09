// Package httperror defines the format for HTTP error responses
// from Chain services.
package httperror

import (
	"context"
	"net/http"

	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
)

// Info contains a set of error codes to send to the user.
type Info struct {
	HTTPStatus int    `json:"-"`
	ChainCode  string `json:"code"`
	Message    string `json:"msg,omitempty"`
}

// Response defines the error response for a Chain error.
type Response struct {
	Info
	Status string                 `json:"status,omitempty"`
	Detail string                 `json:"detail,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// Formatter defines rules for mapping errors to the Chain error
// response format.
type Formatter map[string]Info

// Format builds an error Response body describing err by consulting
// the f lookup table. If no entry is found, it returns f.Default.
func (f Formatter) Format(err error) (body Response) {
	root := errors.Root(err)
	// Some types cannot be used as map keys, for example slices.
	// If an error's underlying type is one of these, don't panic.
	// Just treat it like any other missing entry.
	defer func() {
		if err := recover(); err != nil {
			body = Response{Info{500, "BTM000", "API error"}, "fail", "", nil}
		}
	}()
	info, ok := f[root.Error()]
	if !ok {
		info = Info{500, "BTM000", errors.Detail(err)}
	}

	body = Response{
		Info:   info,
		Status: "fail",
		Detail: errors.Detail(err),
		Data:   errors.Data(err),
	}
	return body
}

// Write writes a json encoded Response to the ResponseWriter.
// It uses the status code associated with the error.
//
// Write may be used as an ErrorWriter in the httpjson package.
func (f Formatter) Write(ctx context.Context, w http.ResponseWriter, err error) {
	resp := f.Format(err)
	httpjson.Write(ctx, w, resp.HTTPStatus, resp)
}
