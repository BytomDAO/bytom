package httpjson

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	berr "github.com/bytom/errors"
)

// ErrorWriter is responsible for writing the provided error value
// to the response.
type ErrorWriter func(context.Context, http.ResponseWriter, error)

// DefaultResponse will be sent as the response body
// when the handler function signature
// has no return value.
var DefaultResponse = json.RawMessage(`{"message":"ok"}`)

// handler is an http.Handler that calls a function for each request.
// It uses the signature of the function to decide how to interpret
type handler struct {
	fv      reflect.Value
	inType  reflect.Type
	hasCtx  bool
	errFunc ErrorWriter
}

// Response describes the response standard.
type Response struct {
	Status string      `json:"status,omitempty"`
	Msg    string      `json:"msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

const (
	// SUCCESS indicates the rpc calling is successful.
	SUCCESS = "success"
	// FAIL indicated the rpc calling is failed.
	FAIL = "fail"
)

// Handler returns an HTTP handler for function f.
// See the package doc for details on allowed signatures for f.
// If f returns a non-nil error, the handler will call errFunc.
func Handler(f interface{}, errFunc ErrorWriter) (http.Handler, error) {
	fv := reflect.ValueOf(f)
	hasCtx, inType, err := funcInputType(fv)
	if err != nil {
		return nil, err
	}

	h := &handler{fv, inType, hasCtx, errFunc}
	return h, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var a []reflect.Value
	if h.hasCtx {
		ctx := req.Context()
		ctx = context.WithValue(ctx, reqKey, req)
		ctx = context.WithValue(ctx, respKey, w)
		a = append(a, reflect.ValueOf(ctx))
	}
	if h.inType != nil {
		inPtr := reflect.New(h.inType)
		err := Read(req.Body, inPtr.Interface())
		if err != nil {
			h.errFunc(req.Context(), w, err)
			return
		}
		a = append(a, inPtr.Elem())
	}
	rv := h.fv.Call(a)

	if len(rv) != 1 {
		h.errFunc(req.Context(), w, errors.New("Exception of response result"))
		return
	}

	result, err := json.Marshal(rv[0].Interface())
	if err != nil {
		h.errFunc(req.Context(), w, berr.WithDetail(err, "json marshal error"))
		return
	}

	resp := Response{}
	if err := json.Unmarshal(result, &resp); err != nil {
		h.errFunc(req.Context(), w, berr.WithDetail(err, "json unmarshal error"))
		return
	}

	if resp.Status == FAIL {
		// restore error message to bytom errors struct
		errSplits := strings.Split(resp.Msg, ": ")
		rootErr := errSplits[len(errSplits)-1]
		detailErr := errSplits[:len(errSplits)-1]

		var detail string
		for i, s := range detailErr {
			if i == 0 {
				detail = detail + s
				continue
			}
			detail = detail + ": " + s
		}

		err := berr.New(rootErr)
		err = berr.WithDetail(err, detail)
		h.errFunc(req.Context(), w, err)
		return
	}

	Write(req.Context(), w, 200, rv[0].Interface())
}

var (
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
)

func funcInputType(fv reflect.Value) (hasCtx bool, t reflect.Type, err error) {
	ft := fv.Type()
	if ft.Kind() != reflect.Func || ft.IsVariadic() {
		return false, nil, errors.New("need nonvariadic func in " + ft.String())
	}

	off := 0 // or 1 with context
	hasCtx = ft.NumIn() >= 1 && ft.In(0).Implements(contextType)
	if hasCtx {
		off = 1
	}

	if ft.NumIn() > off+1 {
		return false, nil, errors.New("too many params in " + ft.String())
	}

	if ft.NumIn() == off+1 {
		t = ft.In(ft.NumIn() - 1)
	}

	if n := ft.NumOut(); n == 2 && !ft.Out(1).Implements(errorType) {
		return false, nil, errors.New("second return value must be error in " + ft.String())
	} else if n > 2 {
		return false, nil, errors.New("need at most two return values in " + ft.String())
	}

	return hasCtx, t, nil
}
