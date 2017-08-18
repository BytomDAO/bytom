package errors_test

import "bytom/errors"

var ErrInvalidKey = errors.New("invalid key")

func ExampleSub() {
	err := sign()
	if err != nil {
		err = errors.Sub(ErrInvalidKey, err)
		return
	}
}

func ExampleSub_return() {
	err := sign()
	err = errors.Sub(ErrInvalidKey, err)
	return
}

func sign() error { return nil }
