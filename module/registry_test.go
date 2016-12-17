package module

import (
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func TestRegister(t *testing.T) {
	name := "UltraMegaSuperCrusher"
	expected := &mockPatcher{}

	Register(name, func(c Config) (Patcher, error) {
		expected.value = c["key"].(string)
		return expected, nil
	})

	init, err := Lookup(name)
	assert.Equal(t, err, nil)

	p, err := init(Config{"key": "hello"})
	assert.Equal(t, err, nil)
	assert.Equal(t, expected, p)

	_, err = Lookup("unknown")
	assert.NotEqual(t, err, nil)
}

type mockPatcher struct {
	value string
}

func (p mockPatcher) Patch(string, interface{}) error { return nil }
func (p mockPatcher) Output(string) (*Out, error)     { return nil, nil }
func (p mockPatcher) Reset() error                    { return nil }
