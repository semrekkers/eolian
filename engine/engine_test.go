package engine

import (
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func TestLifecycle(t *testing.T) {
	e, err := New(1)
	assert.Equal(t, err, nil)

	go e.Run()
	go func() {
		for err := range e.Errors() {
			t.Error(err)
		}
	}()

	err = e.Stop()
	assert.Equal(t, err, nil)
}

func TestInvalidOutputID(t *testing.T) {
	_, err := New(100)
	assert.NotEqual(t, err, nil)
}
