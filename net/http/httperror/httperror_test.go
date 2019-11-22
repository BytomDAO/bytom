package httperror

import (
	"context"
	"fmt"
	"testing"

	"github.com/bytom/bytom/errors"
)

var (
	errNotFound   = errors.New("not found")
	testFormatter = Formatter{
		Default:     Info{500, "CH000", "Internal server error"},
		IsTemporary: func(Info, error) bool { return false },
		Errors: map[error]Info{
			errNotFound: {400, "CH002", "Not found"},
		},
	}
)

// Dummy error type, to test that Format
// doesn't panic when it's used as a map key.
type sliceError []int

func (err sliceError) Error() string { return "slice error" }

func TestInfo(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 500},
		{context.Canceled, 500},
		{errNotFound, 400},
		{errors.Wrap(errNotFound, "foo"), 400},
		{sliceError{}, 500},
		{fmt.Errorf("an error!"), 500},
	}

	for _, test := range cases {
		resp := testFormatter.Format(test.err)
		got := resp.HTTPStatus
		if got != test.want {
			t.Errorf("errInfo(%#v) = %d want %d", test.err, got, test.want)
		}
	}
}
