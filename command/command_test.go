package command

import (
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func TestRun(t *testing.T) {
	err := Run([]string{})
	assert.Equal(t, err, nil)

	err = Run([]string{"../racks/template.lua"})
	assert.Equal(t, err, nil)

	err = Run([]string{"notexistant.lua"})
	assert.NotEqual(t, err, nil)
}
