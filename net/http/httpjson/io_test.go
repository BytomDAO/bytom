package httpjson

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteArray(t *testing.T) {
	examples := []struct {
		in   []int
		want string
	}{
		{nil, "[]"},
		{[]int{}, "[]"},
		{make([]int, 0), "[]"},
	}

	for _, ex := range examples {
		rec := httptest.NewRecorder()
		Write(context.Background(), rec, 200, ex.in)
		got := strings.TrimSpace(rec.Body.String())
		if got != ex.want {
			t.Errorf("Write(%v) = %v want %v", ex.in, got, ex.want)
		}
	}
}

type errResponse struct {
	*httptest.ResponseRecorder
	err error
}

func (r *errResponse) Write([]byte) (int, error) {
	return 0, r.err
}
