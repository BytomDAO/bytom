package test

import (
	"testing"
)

func TestState(t *testing.T) {
	walk(t, stateTestDir, func(t *testing.T, name string, test *StateTestConfig) {
		if err := test.Run(); err != nil {
			t.Fatal(err)
		}
	})
}
